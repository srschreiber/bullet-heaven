package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	_ "image/png" // PNG decoder

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	// shaders/retro_shader

	_ "embed"
)

//go:embed shaders/retro.kage
var retroShaderSrc []byte

var earthImage *ebiten.Image
var smokeImage *ebiten.Image
var fireImage *ebiten.Image
var heroImage *ebiten.Image

func loadImage(path string) *ebiten.Image {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	return ebiten.NewImageFromImage(img)
}

func Init() {
	const earthImagePath = "assets/earth.png"
	const smokeImagePath = "assets/smoke.png"
	const fireImagePath = "assets/fire.png"
	const heroImagePath = "assets/hero1.png"

	earthImage = loadImage(earthImagePath)
	smokeImage = loadImage(smokeImagePath)
	fireImage = loadImage(fireImagePath)
	heroImage = loadImage(heroImagePath)
}

// walking enum
const (
	HeroAnimIdle      = iota // 0
	HeroAnimWalkRight        // 1
	HeroAnimWalkLeft         // 2
	HeroAnimWalkDown         // 3
)

func getHeroAnim(state int) *ebiten.Image {
	/* The image has 4 frames. We only care about the first row which has
	 	Standing still, walking right.

		If not moving, standing still.
		If moving right, render walking right
		If moving left, render walking right flipped
		If moving down, render walking right for now
	*/
	var subImg *ebiten.Image

	// first get w of image
	w := heroImage.Bounds().Dx()
	if state == HeroAnimIdle {
		subImg = heroImage.SubImage(image.Rect(0, 0, w/2, w/2)).(*ebiten.Image)
	} else if state == HeroAnimWalkRight {
		subImg = heroImage.SubImage(image.Rect(w/2, 0, w, w/2)).(*ebiten.Image)
	} else if state == HeroAnimWalkLeft {
		subImg = heroImage.SubImage(image.Rect(w/2, 0, w, w/2)).(*ebiten.Image)
	} else if state == HeroAnimWalkDown {
		subImg = heroImage.SubImage(image.Rect(16, 0, 32, 16)).(*ebiten.Image)
	}

	return subImg
}

// -------------------- Math / Vec2 --------------------

type Vec2 struct{ X, Y float32 }

var Vec2Zero = &Vec2{0, 0}

func (v *Vec2) Norm() *Vec2 {
	m := math.Hypot(float64(v.X), float64(v.Y))
	if m == 0 {
		return Vec2Zero
	}
	return &Vec2{v.X / float32(m), v.Y / float32(m)}
}
func (v *Vec2) Add(u *Vec2) *Vec2   { return &Vec2{v.X + u.X, v.Y + u.Y} }
func (v *Vec2) Sub(u *Vec2) *Vec2   { return &Vec2{v.X - u.X, v.Y - u.Y} }
func (v *Vec2) Mul(s float32) *Vec2 { return &Vec2{v.X * s, v.Y * s} }
func (v *Vec2) Distance(u *Vec2) float32 {
	return float32(math.Hypot(float64(v.X-u.X), float64(v.Y-u.Y)))
}

// -------------------- Game types --------------------

type Player struct {
	Pos       *Vec2
	Direction *Vec2
	Speed     float32 // pixels per second
	Weapons   []Weapon
}

type Projectile struct {
	Pos    *Vec2
	Dir    *Vec2 // unit direction
	Speed  float32
	Radius float32
}

type Weapon struct {
	CooldownSec        float32
	TimeSinceFire      float32
	Projectiles        []Projectile
	ProjectileInstance *Projectile
	LastDir            *Vec2 // remembers last fire direction if aiming is zero
	ParticleEmitter    *SmokeEmitter
}

type Game struct {
	Player       Player
	ScreenWidth  int
	ScreenHeight int
	retroShader  *ebiten.Shader
	offscreen    *ebiten.Image
	startedAt    time.Time
	off          *ebiten.Image
}

func (v Vec2) IsInBounds(g *Game, buffer int) bool {
	return v.X >= float32(buffer) && v.X < float32(g.ScreenWidth-buffer) &&
		v.Y >= float32(buffer) && v.Y < float32(g.ScreenHeight-buffer)
}

// -------------------- Game loop --------------------

const TargetTPS = 120.0

func (g *Game) Update() error {
	// fixed dt tied to TargetTPS
	dt := float32(1.0 / TargetTPS)

	// aim at cursor (logical coords)
	cursorX, cursorY := ebiten.CursorPosition()
	cursor := &Vec2{float32(cursorX), float32(cursorY)}
	if g.Player.Pos.Distance(cursor) < 5 {
		cursor = g.Player.Pos
	}

	// smooth player movement
	g.Player.Direction = cursor.Sub(g.Player.Pos).Norm()
	vel := g.Player.Direction.Mul(g.Player.Speed * dt)

	half := 8 // half-size of player (16px)
	if g.Player.Pos.Add(vel).IsInBounds(g, half) {
		g.Player.Pos = g.Player.Pos.Add(vel)
	} else if g.Player.Pos.Add(&Vec2{X: vel.X}).IsInBounds(g, half) {
		g.Player.Pos = g.Player.Pos.Add(&Vec2{X: vel.X})
	} else if g.Player.Pos.Add(&Vec2{Y: vel.Y}).IsInBounds(g, half) {
		g.Player.Pos = g.Player.Pos.Add(&Vec2{Y: vel.Y})
	}

	// weapons & projectiles
	for i := range g.Player.Weapons {
		w := &g.Player.Weapons[i]
		w.TimeSinceFire += dt

		// move + cull + emit smoke
		newProjectiles := w.Projectiles[:0]
		for j := range w.Projectiles {
			p := &w.Projectiles[j]

			// integrate motion
			p.Pos = p.Pos.Add(p.Dir.Mul(p.Speed * dt))

			w.ParticleEmitter.EmitDirectional(p.Pos, p.Dir, 2, p.Speed)

			// keep if on-screen
			if p.Pos.IsInBounds(g, 0) {
				newProjectiles = append(newProjectiles, *p)
			}

		}
		w.Projectiles = newProjectiles

		// fire when cooldown elapses
		if w.TimeSinceFire >= w.CooldownSec {
			w.TimeSinceFire = 0 + (rand.Float32()*2-1)*0.1*w.CooldownSec // add some randomness to rate of fire

			newProj := *w.ProjectileInstance
			newProj.Pos = g.Player.Pos

			newProj.Dir = g.Player.Direction.Norm()
			// get last direction
			isMoving := newProj.Dir != Vec2Zero

			if !isMoving {
				newProj.Dir = w.LastDir
			}

			// add some randomness
			randomizedVec := &Vec2{X: (rand.Float32()*2 - 1) * 0.5, Y: (rand.Float32()*2 - 1) * 0.5}
			randomizedVec = randomizedVec.Norm().Mul(.1)
			newProj.Dir = newProj.Dir.Add(randomizedVec).Norm()

			w.Projectiles = append(w.Projectiles, newProj)
			if isMoving {
				w.LastDir = g.Player.Direction.Norm()
			}
		}
		w.ParticleEmitter.Update(dt)
	}

	return nil
}

func (g *Game) drawScene(dst *ebiten.Image) {
	// background
	ebitenutil.DrawRect(dst, 0, 0, float64(g.ScreenWidth), float64(g.ScreenHeight),
		color.RGBA{R: 0, G: 100, B: 200, A: 255})

	// player (16x16 square)
	const w = 16.0
	// ebitenutil.DrawRect(dst, float64(g.Player.Pos.X-w/2), float64(g.Player.Pos.Y-w/2), float64(w), float64(w), color.White)
	// check dir for what direction
	var animSprite *ebiten.Image
	if g.Player.Direction.X > 0 {
		// facing right
		animSprite = getHeroAnim(HeroAnimWalkRight)
	} else if g.Player.Direction.X < 0 {
		// facing left
		animSprite = getHeroAnim(HeroAnimWalkLeft)
	} else if g.Player.Direction.Y > 0 {
		// facing down
		animSprite = getHeroAnim(HeroAnimWalkDown)
	} else {
		animSprite = getHeroAnim(HeroAnimIdle)
	}

	op := &ebiten.DrawImageOptions{}
	// figure out how to scale it to 20
	tgtWidth := float64(20)
	l := animSprite.Bounds().Dx()
	s := float64(tgtWidth) / float64(l)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(-tgtWidth/2, -tgtWidth/2)
	op.GeoM.Translate(float64(g.Player.Pos.X), float64(g.Player.Pos.Y))

	dst.DrawImage(animSprite, op)

	// particles
	for _, w := range g.Player.Weapons {
		w.ParticleEmitter.Draw(dst)
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.off == nil || g.off.Bounds().Dx() != g.ScreenWidth || g.off.Bounds().Dy() != g.ScreenHeight {
		g.off = ebiten.NewImage(g.ScreenWidth, g.ScreenHeight)
	}
	g.off.Clear()
	g.drawScene(g.off) // your current drawing code extracted into a helper

	// pass uniforms
	t := float32(time.Since(g.startedAt).Seconds())

	uniforms := map[string]interface{}{
		"Time":       t,
		"Resolution": []float32{float32(g.ScreenWidth), float32(g.ScreenHeight)},

		// nice VS-ish defaults — tweak live
		"PixelSize":    float32(2),               // 1..4
		"Vignette":     float32(0.4),             // 0..1
		"Grain":        float32(0.07),            // 0..0.4
		"Bloom":        float32(0.55),            // 0..1
		"Aberration":   float32(0.002),           // 0..0.005
		"Saturation":   float32(.8),              // 0.8..1.3
		"Contrast":     float32(1.2),             // 0.9..1.2
		"Gamma":        float32(1.2),             // 0.9..1.4
		"Border":       float32(1.5),             // intensity (try 0.8–1.5)
		"BorderClamp":  float32(.3),              // max darken (0.15–0.35)
		"BorderRadius": float32(.8),              // neighbor distance in px (1–2)
		"BorderTint":   []float32{0.0, 0.0, 0.0}, // black
	}

	op := &ebiten.DrawRectShaderOptions{
		Images:   [4]*ebiten.Image{g.off, g.off, g.off, g.off}, // imageSrc0
		Uniforms: uniforms,
	}
	screen.DrawRectShader(g.ScreenWidth, g.ScreenHeight, g.retroShader, op)

	// debug
	actualTPS := ebiten.CurrentTPS()

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Actual TPS: %f", actualTPS), 10, 10)
	numProjectiles := 0
	numParticles := 0
	for _, w := range g.Player.Weapons {
		numProjectiles += len(w.Projectiles)
		numParticles += len(w.ParticleEmitter.Particles)
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Num Projectiles: %d", numProjectiles), 10, 30)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Num Particles: %.2fK", float32(numParticles)/1000), 10, 50)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	// logical size
	return g.ScreenWidth, g.ScreenHeight
}

func main() {
	Init()

	const (
		logicalW = 320
		logicalH = 240
		scale    = 2
	)

	ebiten.SetWindowSize(logicalW*scale, logicalH*scale)
	ebiten.SetWindowTitle("Smoke Particles Demo")
	ebiten.SetTPS(int(TargetTPS))

	defaultCooldown := float32(2)
	earthProjectile := Projectile{
		Pos:    Vec2Zero,
		Dir:    Vec2Zero,
		Speed:  160, // px/sec
		Radius: 5,
	}

	earthWeapon := Weapon{
		CooldownSec:        defaultCooldown,
		Projectiles:        []Projectile{},
		ProjectileInstance: &earthProjectile,
		LastDir:            &Vec2{0.5, 0.5},
		ParticleEmitter:    NewSmokeEmitter(earthImage, 20000, .1, 1),
		TimeSinceFire:      rand.Float32() * defaultCooldown, // stagger fire times
	}

	fireProjectile := Projectile{
		Pos:    Vec2Zero,
		Dir:    Vec2Zero,
		Speed:  160, // px/sec
		Radius: 5,
	}

	fireWeapon := Weapon{
		CooldownSec:        defaultCooldown,
		Projectiles:        []Projectile{},
		ProjectileInstance: &fireProjectile,
		LastDir:            &Vec2{0.5, 0.5},
		ParticleEmitter:    NewSmokeEmitter(fireImage, 20000, .1, 5),
		TimeSinceFire:      rand.Float32() * defaultCooldown, // stagger fire times
	}

	smokeProjectile := Projectile{
		Pos:    Vec2Zero,
		Dir:    Vec2Zero,
		Speed:  160, // px/sec
		Radius: 5,
	}

	smokeWeapon := Weapon{
		CooldownSec:        defaultCooldown,
		Projectiles:        []Projectile{},
		ProjectileInstance: &smokeProjectile,
		LastDir:            &Vec2{0.5, 0.5},
		ParticleEmitter:    NewSmokeEmitter(smokeImage, 20000, .1, 1),
		TimeSinceFire:      rand.Float32() * defaultCooldown, // stagger fire times
	}

	game := &Game{
		ScreenWidth:  logicalW,
		ScreenHeight: logicalH,
		Player: Player{
			Pos:       &Vec2{X: 100, Y: 100},
			Direction: Vec2Zero,
			Speed:     80, // px/sec
			Weapons:   []Weapon{earthWeapon, fireWeapon, smokeWeapon},
		},
		startedAt: time.Now(),
	}

	sh, err := ebiten.NewShader([]byte(retroShaderSrc))
	if err != nil {
		log.Fatal(err)
	}
	game.retroShader = sh

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
