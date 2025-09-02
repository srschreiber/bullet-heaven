package main

import (
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"

	_ "image/png" // PNG decoder

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var earthImage *ebiten.Image
var smokeImage *ebiten.Image
var fireImage *ebiten.Image

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

	earthImage = loadImage(earthImagePath)
	smokeImage = loadImage(smokeImagePath)
	fireImage = loadImage(fireImagePath)
}

// -------------------- Math / Vec2 --------------------

type Vec2 struct{ X, Y float64 }

var Vec2Zero = Vec2{0, 0}

func (v Vec2) Norm() Vec2 {
	m := math.Hypot(v.X, v.Y)
	if m == 0 {
		return Vec2Zero
	}
	return Vec2{v.X / m, v.Y / m}
}
func (v Vec2) Add(u Vec2) Vec2    { return Vec2{v.X + u.X, v.Y + u.Y} }
func (v Vec2) Sub(u Vec2) Vec2    { return Vec2{v.X - u.X, v.Y - u.Y} }
func (v Vec2) Mul(s float64) Vec2 { return Vec2{v.X * s, v.Y * s} }
func (v Vec2) Distance(u Vec2) float64 {
	return math.Hypot(v.X-u.X, v.Y-u.Y)
}

// -------------------- Game types --------------------

type Player struct {
	Pos       Vec2
	Direction Vec2
	Speed     float64 // pixels per second
	Weapons   []Weapon
}

type Projectile struct {
	Pos    Vec2
	Dir    Vec2 // unit direction
	Speed  float64
	Radius float64
}

type Weapon struct {
	CooldownSec        float64
	TimeSinceFire      float64
	Projectiles        []Projectile
	ProjectileInstance *Projectile
	LastDir            Vec2 // remembers last fire direction if aiming is zero
	ParticleEmitter    *SmokeEmitter
}

type Game struct {
	Player       Player
	ScreenWidth  int
	ScreenHeight int
}

func (v Vec2) IsInBounds(g *Game, buffer int) bool {
	return v.X >= float64(buffer) && v.X < float64(g.ScreenWidth-buffer) &&
		v.Y >= float64(buffer) && v.Y < float64(g.ScreenHeight-buffer)
}

// -------------------- Game loop --------------------

const TargetTPS = 120.0

func (g *Game) Update() error {
	// fixed dt tied to TargetTPS
	dt := 1.0 / TargetTPS

	// aim at cursor (logical coords)
	cursorX, cursorY := ebiten.CursorPosition()
	cursor := Vec2{float64(cursorX), float64(cursorY)}
	if g.Player.Pos.Distance(cursor) < 5 {
		cursor = g.Player.Pos
	}

	// smooth player movement
	g.Player.Direction = cursor.Sub(g.Player.Pos).Norm()
	vel := g.Player.Direction.Mul(g.Player.Speed * dt)

	half := 8 // half-size of player (16px)
	if g.Player.Pos.Add(vel).IsInBounds(g, half) {
		g.Player.Pos = g.Player.Pos.Add(vel)
	} else if g.Player.Pos.Add(Vec2{X: vel.X}).IsInBounds(g, half) {
		g.Player.Pos = g.Player.Pos.Add(Vec2{X: vel.X})
	} else if g.Player.Pos.Add(Vec2{Y: vel.Y}).IsInBounds(g, half) {
		g.Player.Pos = g.Player.Pos.Add(Vec2{Y: vel.Y})
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

			w.ParticleEmitter.EmitDirectional(p.Pos, p.Dir, 3, p.Speed)

			// keep if on-screen
			if p.Pos.IsInBounds(g, 0) {
				newProjectiles = append(newProjectiles, *p)
			}

		}
		w.Projectiles = newProjectiles

		// fire when cooldown elapses
		if w.TimeSinceFire >= w.CooldownSec {
			w.TimeSinceFire = 0

			newProj := *w.ProjectileInstance
			newProj.Pos = g.Player.Pos

			newProj.Dir = g.Player.Direction.Norm()
			// get last direction
			isMoving := newProj.Dir != Vec2Zero

			if !isMoving {
				newProj.Dir = w.LastDir
			}

			// add some randomness
			randomizedVec := Vec2{X: (rand.Float64()*2 - 1) * 0.5, Y: (rand.Float64()*2 - 1) * 0.5}
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

func (g *Game) Draw(screen *ebiten.Image) {
	// background
	ebitenutil.DrawRect(screen, 0, 0, float64(g.ScreenWidth), float64(g.ScreenHeight),
		color.RGBA{R: 128, G: 0, B: 128, A: 255})

	// player (16x16 square)
	const w = 16.0
	ebitenutil.DrawRect(screen, g.Player.Pos.X-w/2, g.Player.Pos.Y-w/2, w, w, color.White)

	// projectiles (simple circles)
	// for i := range g.Player.Weapons {
	// 	w := g.Player.Weapons[i]
	// 	for _, proj := range w.Projectiles {
	// 		ebitenutil.DrawCircle(screen, proj.Pos.X, proj.Pos.Y, proj.Radius, color.RGBA{R: 255, A: 255})
	// 	}
	// }

	// draw all weapon art
	for _, w := range g.Player.Weapons {
		w.ParticleEmitter.Draw(screen)
	}
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

	earthProjectile := Projectile{
		Pos:    Vec2Zero,
		Dir:    Vec2Zero,
		Speed:  160, // px/sec
		Radius: 5,
	}

	earthWeapon := Weapon{
		CooldownSec:        .1,
		Projectiles:        []Projectile{},
		ProjectileInstance: &earthProjectile,
		LastDir:            Vec2{0.5, 0.5},
		ParticleEmitter:    NewSmokeEmitter(earthImage, 20000, .1, 1),
	}

	fireProjectile := Projectile{
		Pos:    Vec2Zero,
		Dir:    Vec2Zero,
		Speed:  160, // px/sec
		Radius: 5,
	}

	fireWeapon := Weapon{
		CooldownSec:        .1,
		Projectiles:        []Projectile{},
		ProjectileInstance: &fireProjectile,
		LastDir:            Vec2{0.5, 0.5},
		ParticleEmitter:    NewSmokeEmitter(fireImage, 20000, .1, 5),
	}

	smokeProjectile := Projectile{
		Pos:    Vec2Zero,
		Dir:    Vec2Zero,
		Speed:  160, // px/sec
		Radius: 5,
	}

	smokeWeapon := Weapon{
		CooldownSec:        .1,
		Projectiles:        []Projectile{},
		ProjectileInstance: &smokeProjectile,
		LastDir:            Vec2{0.5, 0.5},
		ParticleEmitter:    NewSmokeEmitter(smokeImage, 20000, .1, 1),
	}

	game := &Game{
		ScreenWidth:  logicalW,
		ScreenHeight: logicalH,
		Player: Player{
			Pos:       Vec2{X: 100, Y: 100},
			Direction: Vec2Zero,
			Speed:     80, // px/sec
			Weapons:   []Weapon{earthWeapon, fireWeapon, smokeWeapon},
		},
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
