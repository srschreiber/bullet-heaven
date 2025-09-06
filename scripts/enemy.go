package scripts

import (
	"math/rand"
	"time"
)

var AllEnemies []*Enemy

type Collider struct {
	radius         float32
	offsetPosition *Vec2
}

type Enemy struct {
	Pos             *Vec2
	Direction       *Vec2
	Speed           float32 // pixels per second
	Weapons         []Weapon
	MaxHealth       rune
	Health          rune
	RespawnCooldown rune
	RespawnTimer    rune
	WalkAnimator    *WalkingAnimationManager
	OriginalPos     *Vec2
	Name            string
	AggroRadius     float32
	RandomOffset    *Vec2
	Width           float32
	Colliders       []Collider
}

func NewSkeletonEnemy(pos *Vec2) *Enemy {

	newEnemy := &Enemy{
		Pos:             pos,
		OriginalPos:     &Vec2{X: pos.X, Y: pos.Y},
		Direction:       &Vec2{X: 0, Y: 0},
		Speed:           50,
		Weapons:         []Weapon{},
		MaxHealth:       100,
		Health:          100,
		RespawnCooldown: 5,
		RespawnTimer:    0,
		WalkAnimator:    NewCharacterWalkingAnimator("assets/enemies/skeletonspritesheet.png"),
		Name:            "Skeleton",
		AggroRadius:     500,
		// so all enemies don't flock to same place
		RandomOffset: &(Vec2{X: float32(rand.Intn(2)) - 1, Y: float32(rand.Intn(2)) - 1}),
		Width:        64,
	}

	// have a few colliders, 3 top, 3 middle, 3 bottom

	leftColliderX := float32(-1 * 8)
	topColliderY := float32(-1 * 8)

	colliderGapX := float32(1 * 8)
	colliderGapY := newEnemy.Width / 5

	colliders := []Collider{}
	// top colliders
	colliderRadius := float32(10)
	colliders = append(colliders, Collider{radius: colliderRadius, offsetPosition: &Vec2{X: leftColliderX, Y: topColliderY}})
	colliders = append(colliders, Collider{radius: colliderRadius, offsetPosition: &Vec2{X: leftColliderX + colliderGapX, Y: topColliderY}})
	colliders = append(colliders, Collider{radius: colliderRadius, offsetPosition: &Vec2{X: leftColliderX + colliderGapX*2, Y: topColliderY}})

	// middle
	colliders = append(colliders, Collider{radius: colliderRadius, offsetPosition: &Vec2{X: leftColliderX, Y: topColliderY + colliderGapY}})
	colliders = append(colliders, Collider{radius: colliderRadius, offsetPosition: &Vec2{X: leftColliderX + colliderGapX, Y: topColliderY + colliderGapY}})
	colliders = append(colliders, Collider{radius: colliderRadius, offsetPosition: &Vec2{X: leftColliderX + colliderGapX*2, Y: topColliderY + colliderGapY}})

	// bottom
	colliders = append(colliders, Collider{radius: colliderRadius, offsetPosition: &Vec2{X: leftColliderX, Y: topColliderY + colliderGapY*2}})
	colliders = append(colliders, Collider{radius: colliderRadius, offsetPosition: &Vec2{X: leftColliderX + colliderGapX, Y: topColliderY + colliderGapY*2}})
	colliders = append(colliders, Collider{radius: colliderRadius, offsetPosition: &Vec2{X: leftColliderX + colliderGapX*2, Y: topColliderY + colliderGapY*2}})

	newEnemy.Colliders = colliders
	AllEnemies = append(AllEnemies, newEnemy)
	return newEnemy
}

func (e *Enemy) IsDead() bool {
	return e.Health <= 0
}

func (e *Enemy) Update(dt float32, player *Player) {
	dtMs := time.Duration(dt*1000) * time.Millisecond

	surroundingProjectiles := player.ProjectileGrid.GetSurroundingProjectiles(e.Pos, int(e.Width)*2)
	// hack, get raw list of projectiles from player weapons
	// var surroundingProjectiles []*Projectile
	// for _, weapon := range player.Weapons {
	// 	surroundingProjectiles = append(surroundingProjectiles, weapon.Projectiles...)
	// }

	knockbackVector := &Vec2{X: 0, Y: 0}
	for _, proj := range surroundingProjectiles {
		// check if close to any collider within its radius
		for _, collider := range e.Colliders {
			if proj.Pos.Distance(e.Pos.Add(collider.offsetPosition)) <= collider.radius {
				// Handle collision
				knockbackVector = knockbackVector.Add(proj.Dir)
				e.Health -= 1
				break
			}
		}
	}

	if e.IsDead() {
		return
	}

	if e.Pos.Distance(player.Pos) <= e.AggroRadius && e.Pos.Distance(player.Pos) > player.Width/2 {
		var targetDest = player.Pos.Add(e.RandomOffset.Mul(player.Width / 4))
		var moveDirection *Vec2 = &Vec2{
			X: float32(targetDest.X - e.Pos.X),
			Y: float32(targetDest.Y - e.Pos.Y),
		}
		moveDirection = moveDirection.Norm()

		vel := moveDirection.Mul(e.Speed * dt).Add(knockbackVector)
		e.Pos = e.Pos.Add(vel)
		e.WalkAnimator.UpdateByDirection(float64(moveDirection.X), float64(moveDirection.Y), dtMs, true, "")
	} else if e.Pos.Distance(player.Pos) <= player.Width/2 {
		// stop moving
		//e.WalkAnimator.UpdateByDirection(0, 0, dtMs, false, "")
		// Attack Animation: TODO!
	}
}
