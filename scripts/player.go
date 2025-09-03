package scripts

import (
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type Player struct {
	Pos               *Vec2
	Direction         *Vec2
	Speed             float32 // pixels per second
	Weapons           []Weapon
	MaxHealth         rune
	Health            rune
	MaxMana           rune
	Mana              rune
	ManaRegenRate     float32 // mana per second
	ManaRegenCooldown time.Duration
	StrifeDuration    float32 // length
	StrifeTime        float32
	LastStrife        time.Time
	StrifeCooldown    time.Duration
	StrifeMultiplier  float32
	StrifeDecay       float32 // loss of speed
}

func (p *Player) Update(dt float32) {
	cursorX, cursorY := ebiten.CursorPosition()
	cursor := &Vec2{float32(cursorX), float32(cursorY)}
	if p.Pos.Distance(cursor) < 5 {
		cursor = p.Pos
	}

	// smooth player movement
	p.Direction = cursor.Sub(p.Pos).Norm()
	vel := p.Direction.Mul(p.Speed * dt)

	// Tick/update
	now := time.Now()

	// 1) Update current strife state first
	if p.StrifeTime > 0 {
		p.StrifeTime -= dt
		timeInStrife := p.StrifeDuration - p.StrifeTime

		mult := p.StrifeMultiplier - timeInStrife*p.StrifeDecay
		if mult < 1 {
			mult = 1
		}
		newVel := vel.Mul(mult)
		vel = newVel

		// detect end-of-strife transition
		if p.StrifeTime <= 0 {
			p.StrifeTime = 0
			p.LastStrife = now
		}
	} else {
		p.StrifeTime = 0 // clamp (in case it went negative)
	}

	// 2) After updating, check if we can start a new strife
	// Prefer edge-trigger to avoid hold-to-retrigger
	if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) &&
		now.After(p.LastStrife.Add(p.StrifeCooldown)) &&
		p.StrifeTime == 0 {
		p.StrifeTime = p.StrifeDuration
	}

	half := 16
	if p.Pos.Add(vel).IsInBounds(GameInstance.ScreenHeight, GameInstance.ScreenWidth, half) {
		p.Pos = p.Pos.Add(vel)
	} else if p.Pos.Add(&Vec2{X: vel.X}).IsInBounds(GameInstance.ScreenHeight, GameInstance.ScreenWidth, half) {
		p.Pos = p.Pos.Add(&Vec2{X: vel.X})
	} else if p.Pos.Add(&Vec2{Y: vel.Y}).IsInBounds(GameInstance.ScreenHeight, GameInstance.ScreenWidth, half) {
		p.Pos = p.Pos.Add(&Vec2{Y: vel.Y})
	}

	// weapons & projectiles
	shot := false
	for i := range p.Weapons {
		w := &p.Weapons[i]
		w.TimeSinceFire += dt

		// move + cull + emit smoke
		newProjectiles := w.Projectiles[:0]
		for j := range w.Projectiles {
			p := &w.Projectiles[j]

			// integrate motion
			p.Pos = p.Pos.Add(p.Dir.Mul(p.Speed * dt))

			w.ParticleEmitter.EmitDirectional(p.Pos, p.Dir, 2, p.Speed)

			// keep if on-screen
			if p.Pos.IsInBounds(GameInstance.ScreenHeight, GameInstance.ScreenWidth, 0) {
				newProjectiles = append(newProjectiles, *p)
			}

		}
		w.Projectiles = newProjectiles

		// fire when cooldown elapses if holding mouse button
		hasMana := statusBarAnimationManager.HasHearts("mana")
		if hasMana && w.TimeSinceFire >= w.CooldownSec && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			w.TimeSinceFire = 0 + (rand.Float32()*2-1)*0.1*w.CooldownSec // add some randomness to rate of fire
			shot = true
			newProj := *w.ProjectileInstance
			newProj.Pos = p.Pos

			newProj.Dir = p.Direction.Norm()
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
				w.LastDir = p.Direction.Norm()
			}
		}
		w.ParticleEmitter.Update(dt)
	}

	if shot {
		statusBarAnimationManager.DecrementHeart(1, "mana")
	}

	p.ManaRegenCooldown -= time.Duration(dt*1000) * time.Millisecond
	if p.ManaRegenCooldown <= 0 {
		p.ManaRegenCooldown = time.Duration(1000/p.ManaRegenRate) * time.Millisecond
		statusBarAnimationManager.IncrementHeart(1, "mana")
	}

	heroAnimationManager.UpdateByDirection(float64(p.Direction.X), float64(p.Direction.Y), time.Duration(dt*1000)*time.Millisecond)

	return
}
