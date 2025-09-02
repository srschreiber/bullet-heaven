package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Vec2 struct {
	X, Y float64
}

var (
	Vec2Zero = Vec2{X: 0, Y: 0}
)

func (v Vec2) Norm() Vec2 {
	m := math.Hypot(v.X, v.Y)
	if m == 0 {
		return Vec2Zero
	}
	return Vec2{v.X / m, v.Y / m}
}

func (v Vec2) Add(u Vec2) Vec2    { return Vec2{X: v.X + u.X, Y: v.Y + u.Y} }
func (v Vec2) Sub(u Vec2) Vec2    { return Vec2{X: v.X - u.X, Y: v.Y - u.Y} }
func (v Vec2) Mul(s float64) Vec2 { return Vec2{X: v.X * s, Y: v.Y * s} }

func (v Vec2) Distance(u Vec2) float64 {
	dx, dy := v.X-u.X, v.Y-u.Y
	return math.Hypot(dx, dy)
}

type Player struct {
	Pos       Vec2
	Direction Vec2
	Speed     float64 // pixels per second
	Weapons   []Weapon
}

type Projectile struct {
	Pos    Vec2
	Dir    Vec2 // should be unit length
	Speed  float64
	Radius float64
}

type Weapon struct {
	CooldownSec        float64
	TimeSinceFire      float64
	Projectiles        []Projectile
	ProjectileInstance *Projectile
	LastDir            Vec2 // useful if not moving
}

func (v Vec2) IsInBounds(g *Game, buffer int) bool {
	return v.X >= float64(buffer) && v.X < float64(g.ScreenWidth-buffer) &&
		v.Y >= float64(buffer) && v.Y < float64(g.ScreenHeight-buffer)
}

type Game struct {
	Player       Player
	ScreenWidth  int
	ScreenHeight int
}

const TargetTPS = 120.0

func (g *Game) Update() error {
	// dt based on actual tick rate
	tps := TargetTPS
	fmt.Println(tps)
	dt := 1.0 / 60.0
	if tps > 0 {
		dt = 1.0 / tps
	}

	// aim at cursor
	cursorX, cursorY := ebiten.CursorPosition()
	cursor := Vec2{X: float64(cursorX), Y: float64(cursorY)}
	if g.Player.Pos.Distance(cursor) < 5 {
		cursor = g.Player.Pos
	}

	// move player smoothly
	g.Player.Direction = cursor.Sub(g.Player.Pos).Norm()
	velocity := g.Player.Direction.Mul(g.Player.Speed * dt)
	if g.Player.Pos.Add(velocity).IsInBounds(g, 16/2) {
		g.Player.Pos = g.Player.Pos.Add(velocity)
	} else if g.Player.Pos.Add(Vec2{X: velocity.X, Y: 0}).IsInBounds(g, 16/2) {
		g.Player.Pos = g.Player.Pos.Add(Vec2{X: velocity.X, Y: 0})
	} else if g.Player.Pos.Add(Vec2{X: 0, Y: velocity.Y}).IsInBounds(g, 16/2) {
		g.Player.Pos = g.Player.Pos.Add(Vec2{X: 0, Y: velocity.Y})
	}

	// weapons & projectiles
	for i := range g.Player.Weapons {
		w := &g.Player.Weapons[i]
		w.TimeSinceFire += dt

		// move existing projectiles (no per-frame re-normalize)
		newProjectiles := w.Projectiles[:0]
		for j := range w.Projectiles {
			p := &w.Projectiles[j]
			p.Pos = p.Pos.Add(p.Dir.Mul(p.Speed * dt))
			if p.Pos.IsInBounds(g, 0) {
				newProjectiles = append(newProjectiles, *p)
			} else {
				fmt.Println("removed", p.Pos)
			}
		}
		w.Projectiles = newProjectiles

		// fire when cooldown elapses
		if w.TimeSinceFire >= w.CooldownSec {
			w.TimeSinceFire = 0

			newProj := *w.ProjectileInstance // struct copy
			newProj.Pos = g.Player.Pos
			newProj.Dir = g.Player.Direction.Norm() // set once
			if newProj.Dir == Vec2Zero {
				newProj.Dir = w.LastDir
			}
			w.Projectiles = append(w.Projectiles, newProj)
			w.LastDir = newProj.Dir
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// player
	const w = 16.0
	ebitenutil.DrawRect(screen, g.Player.Pos.X-w/2, g.Player.Pos.Y-w/2, w, w, color.White)

	// projectiles
	for i := range g.Player.Weapons {
		w := g.Player.Weapons[i]
		for _, proj := range w.Projectiles {
			ebitenutil.DrawCircle(screen, proj.Pos.X, proj.Pos.Y, proj.Radius, color.RGBA{R: 255, A: 255})
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 320 * 2, 240 * 2
}

func main() {
	ScreenHeight := 320 * 2
	ScreenWidth := 240 * 2

	ebiten.SetWindowSize(int(ScreenWidth), int(ScreenHeight))
	ebiten.SetWindowTitle("Smooth Movement Demo")
	ebiten.SetTPS(TargetTPS) // try 60 or 120; both will be smooth with dt

	defaultProjectile := Projectile{
		Pos:    Vec2Zero,
		Dir:    Vec2Zero,
		Speed:  160, // px/sec
		Radius: 5,
	}
	defaultWeapon := Weapon{
		CooldownSec:        0.25, // fire 4/s
		Projectiles:        []Projectile{},
		ProjectileInstance: &defaultProjectile,
		LastDir:            Vec2{X: 0.5, Y: 0.5},
	}

	game := &Game{
		ScreenWidth:  ScreenWidth,
		ScreenHeight: ScreenHeight,
		Player: Player{
			Pos:       Vec2{X: 100, Y: 100},
			Direction: Vec2Zero,
			Speed:     80, // px/sec
			Weapons:   []Weapon{defaultWeapon},
		},
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
