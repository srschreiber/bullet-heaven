package main

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// =======================
// Directional Smoke / Star Trail
// =======================

type SmokeParticle struct {
	Pos   Vec2
	Vel   Vec2
	Life  float64 // remaining lifetime
	Max   float64 // initial lifetime
	Scale float64
	Rot   float64
	Spin  float64
}

type SmokeEmitter struct {
	Img          *ebiten.Image
	Particles    []SmokeParticle
	MaxParticles int

	// Tunables for "shooting star" feel
	ScaleBase  float64 // base starting scale
	ScaleVar   float64 // random extra start scale
	Growth     float64 // scale growth per second
	Damping    float64 // velocity damping per tick (e.g., 0.99)
	SpinRange  float64 // max abs spin (rad/s)
	AlphaCurve int     // 1=linear, 2=quad, 3=cubic

	// Directional trail settings
	Spread   float64 // radians half-angle (e.g., 0.15)
	Jitter   float64 // spawn jitter in px (e.g., 0.5)
	Lifetime float64
}

// NewSmokeEmitter creates a trail-style emitter with sensible defaults.
func NewSmokeEmitter(img *ebiten.Image, max int, scale float64, lifetime float64) *SmokeEmitter {
	return &SmokeEmitter{
		Img:          img,
		Particles:    make([]SmokeParticle, 0, max),
		MaxParticles: max,

		// shooting-star defaults (tight, directional)
		ScaleBase:  scale * 0.28,
		ScaleVar:   scale * 0.10,
		Growth:     -0.03, // shrink
		Damping:    0.95,  // gentle damping to slow down
		SpinRange:  0.25,  // subtle rotation
		AlphaCurve: 3,     // cubic fade

		Spread:   0.05, // ~±10 degrees
		Jitter:   0.5,  // sub-pixel to ~1px
		Lifetime: lifetime,
	}
}

// Emit keeps for compatibility (still works, but directional is preferred).
// Emits with zero forward bias (randomized in a narrow cone around +X).
func (e *SmokeEmitter) Emit(pos Vec2, n int) {
	// default forward dir = +X
	e.EmitDirectional(pos, Vec2{1, 0}, n, 1.0)
}

// EmitDirectional spawns N particles forward along `dir` with a narrow spread.
// `dir` should be normalized; `speedScale` lets you tie speed to projectile speed.
func (e *SmokeEmitter) EmitDirectional(pos Vec2, dir Vec2, n int, speedScale float64) {
	if e.Img == nil || n <= 0 {
		return
	}
	if len(e.Particles) >= e.MaxParticles {
		return
	}

	// ensure direction
	d := dir.Norm()
	base := math.Atan2(d.Y, d.X)

	// clamp how many we can add without realloc
	space := e.MaxParticles - len(e.Particles)
	if n > space {
		n = space
	}

	for i := 0; i < n; i++ {
		// angle within a narrow cone
		ang := base + (rand.Float64()*2-1)*e.Spread
		spd := speedScale + rand.Float64()*speedScale*0.5

		vx := math.Cos(ang) * spd
		vy := math.Sin(ang) * spd

		// slight jitter to avoid perfect overlap
		jx := (rand.Float64()*2 - 1) * e.Jitter
		jy := (rand.Float64()*2 - 1) * e.Jitter

		// lifetime: tight/starry trails look good with shorter life
		life := e.Lifetime + e.Lifetime*rand.Float64()*0.5

		startScale := e.ScaleBase + rand.Float64()*e.ScaleVar
		spin := (rand.Float64()*2 - 1) * e.SpinRange

		e.Particles = append(e.Particles, SmokeParticle{
			Pos:   Vec2{pos.X + jx, pos.Y + jy},
			Vel:   Vec2{vx, vy},
			Life:  float64(life),
			Max:   float64(life),
			Scale: startScale,
			Rot:   0,
			Spin:  spin,
		})
	}
}

// Update advances all particles and culls dead ones.
func (e *SmokeEmitter) Update(dt float64) {
	if len(e.Particles) == 0 {
		return
	}
	next := e.Particles[:0]
	damp := e.Damping
	grow := e.Growth

	for i := 0; i < len(e.Particles); i++ {
		p := e.Particles[i]
		p.Life -= float64(dt)
		if p.Life <= 0 {
			continue
		}

		// integrate
		p.Pos = p.Pos.Add(p.Vel.Mul(dt))

		// keep tight: small damping prevents wide spreading
		p.Vel.X *= damp
		p.Vel.Y *= damp

		// gentle growth + spin
		p.Scale += grow * dt
		p.Scale = math.Max(0, p.Scale)
		p.Rot += p.Spin * dt

		next = append(next, p)
	}
	e.Particles = next
}

// Draw renders all smoke particles with a configurable alpha curve.
func (e *SmokeEmitter) Draw(screen *ebiten.Image) {
	if e.Img == nil || len(e.Particles) == 0 {
		return
	}
	w, h := e.Img.Size()
	for i := 0; i < len(e.Particles); i++ {
		p := &e.Particles[i]

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
		op.GeoM.Rotate(p.Rot)
		op.GeoM.Scale(p.Scale, p.Scale)
		op.GeoM.Translate(p.Pos.X, p.Pos.Y)

		// fade alpha by life with chosen curve
		elapsed := 1.0 - (p.Life / p.Max) // 0..1
		a := 1.0
		switch e.AlphaCurve {
		case 1: // linear
			a = 1.0 - elapsed
		case 2: // quadratic
			a = 1.0 - elapsed*elapsed
		default: // cubic
			a = 1.0 - elapsed*elapsed*elapsed
		}
		if a < 0 {
			a = 0
		}
		op.ColorScale.Scale(1, 1, 1, float32(a))

		screen.DrawImage(e.Img, op)
	}
}
