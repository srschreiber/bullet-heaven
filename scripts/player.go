package scripts

import (
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type Player struct {
	Pos               *Vec2
	MoveDirection     *Vec2
	AimDirection      *Vec2
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
	Width             float32
}

func (p *Player) Update(dt float32) {
	cursorX, cursorY := ebiten.CursorPosition()
	cursor := &Vec2{float32(cursorX), float32(cursorY)}
	if p.Pos.Distance(cursor) < 5 {
		cursor = p.Pos
	}

	p.AimDirection = cursor.Sub(p.Pos).Norm()

	// smooth player movement
	//p.Direction = cursor.Sub(p.Pos).Norm()
	moveDir := &Vec2{X: 0, Y: 0}
	// get directions from wasd
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		moveDir.Y = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		moveDir.Y = 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		moveDir.X = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		moveDir.X = 1
	}

	moveDir = moveDir.Norm()
	p.MoveDirection = moveDir

	vel := p.MoveDirection.Mul(p.Speed * dt)

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
	if ebiten.IsKeyPressed(ebiten.KeySpace) &&
		now.After(p.LastStrife.Add(p.StrifeCooldown)) &&
		p.StrifeTime == 0 {
		p.StrifeTime = p.StrifeDuration
	}

	cursorDistance := p.Pos.Distance(cursor)
	slowZone := float32(100.0)
	// slow down vel as approach cursor
	if cursorDistance < slowZone {
		vel = vel.Mul(float32(cursorDistance) / slowZone)
	}

	if p.Pos.Add(vel).IsInBounds(GameInstance.ScreenWidth, GameInstance.ScreenHeight, int(p.Width/4)) {
		p.Pos = p.Pos.Add(vel)
	} else if p.Pos.Add(&Vec2{X: vel.X}).IsInBounds(GameInstance.ScreenWidth, GameInstance.ScreenHeight, int(p.Width/4)) {
		p.Pos = p.Pos.Add(&Vec2{X: vel.X})
	} else if p.Pos.Add(&Vec2{Y: vel.Y}).IsInBounds(GameInstance.ScreenWidth, GameInstance.ScreenHeight, int(p.Width/4)) {
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
			if p.Pos.IsInBounds(GameInstance.ScreenWidth, GameInstance.ScreenHeight, 0) && p.Gas > 0 {
				newProjectiles = append(newProjectiles, *p)
			}
			p.Gas -= dt * p.Speed
		}
		w.Projectiles = newProjectiles

		// fire when cooldown elapses if holding mouse button
		hasMana := statusBarAnimationManager.HasHearts("mana")

		if hasMana && w.TimeSinceFire >= w.CooldownSec && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			w.TimeSinceFire = 0 + (rand.Float32()*2-1)*0.1*w.CooldownSec // add some randomness to rate of fire
			shot = true
			newProj := *w.ProjectileInstance
			newProj.Pos = p.Pos.Add(p.MoveDirection.Mul(32))

			newProj.Dir = p.AimDirection.Norm()

			// add some randomness
			randomizedVec := &Vec2{X: (rand.Float32()*2 - 1) * 0.5, Y: (rand.Float32()*2 - 1) * 0.5}
			randomizedVec = randomizedVec.Norm().Mul(.1)
			newProj.Dir = newProj.Dir.Add(randomizedVec).Norm()

			w.Projectiles = append(w.Projectiles, newProj)
			w.LastDir = p.AimDirection.Norm()

		}
		w.ParticleEmitter.Update(dt)
	}

	if shot {
		statusBarAnimationManager.DecrementHeart(1, "mana")
	}

	// check if weapon is still in cooldown. If so, can't recover mana
	inWeaponCooldown := false
	for i := range p.Weapons {
		w := &p.Weapons[i]
		if w.TimeSinceFire < w.CooldownSec {
			inWeaponCooldown = true
		}
	}

	if !inWeaponCooldown && !shot {
		// If mouse isnt down, can regen
		p.ManaRegenCooldown -= time.Duration(dt*1000) * time.Millisecond
		if p.ManaRegenCooldown <= 0 {
			p.ManaRegenCooldown = time.Duration(1000/p.ManaRegenRate) * time.Millisecond
			statusBarAnimationManager.IncrementHeart(1, "mana")
		}
	}

	moving := p.MoveDirection.Length() > 0
	if p.StrifeTime > 0 {
		heroAnimationManager.UpdateByDirection(float64(p.AimDirection.X), float64(p.AimDirection.Y), time.Duration(dt*1000)*time.Millisecond, true, true)
	} else {
		heroAnimationManager.UpdateByDirection(float64(p.AimDirection.X), float64(p.AimDirection.Y), time.Duration(dt*1000)*time.Millisecond, false, moving)
	}

	return
}
