package scripts

import "game/model"

type Projectile struct {
	Pos    *model.Vec2
	Dir    *model.Vec2 // unit direction
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
