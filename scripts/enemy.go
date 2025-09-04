package scripts

import (
	"math/rand"
	"time"
)

var AllEnemies []*Enemy

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
		WalkAnimator:    NewCharacterWalkingAnimator("assets/enemies/skeletonspritesheet.png", 8),
		Name:            "Skeleton",
		AggroRadius:     500,
		// so all enemies don't flock to same place
		RandomOffset: &(Vec2{X: float32(rand.Intn(2)) - 1, Y: float32(rand.Intn(2)) - 1}),
		Width:        64,
	}

	AllEnemies = append(AllEnemies, newEnemy)
	return newEnemy
}

func (e *Enemy) Update(dt float32, player *Player) {
	dtMs := time.Duration(dt*1000) * time.Millisecond
	if e.Pos.Distance(player.Pos) <= e.AggroRadius && e.Pos.Distance(player.Pos) > player.Width/2 {
		var targetDest = player.Pos.Add(e.RandomOffset.Mul(player.Width / 4))
		var moveDirection *Vec2 = &Vec2{
			X: float32(targetDest.X - e.Pos.X),
			Y: float32(targetDest.Y - e.Pos.Y),
		}
		moveDirection = moveDirection.Norm()

		vel := moveDirection.Mul(e.Speed * dt)
		e.Pos = e.Pos.Add(vel)
		e.WalkAnimator.UpdateByDirection(float64(moveDirection.X), float64(moveDirection.Y), dtMs, false, true)
	} else if e.Pos.Distance(player.Pos) <= player.Width/2 {
		// stop moving
		e.WalkAnimator.UpdateByDirection(0, 0, dtMs, false, false)
		// Attack
	}
}
