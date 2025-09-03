package scripts

import (
	"fmt"
	"game/model"
	"game/util"
	"image"
	"image/color"
	"log"
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
var heroAnimationManager *util.AnimationManager
var statusBarAnimationManager *util.StatusBarAnimationManager

var heroImagePath = "assets/characters/wizard/standard/walk.png"

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

// -------------------- Game types --------------------
type Vec2 = model.Vec2

var Vec2Zero = model.Vec2Zero

type Game struct {
	Player       Player
	ScreenWidth  int
	ScreenHeight int
	retroShader  *ebiten.Shader
	offscreen    *ebiten.Image
	startedAt    time.Time
	off          *ebiten.Image
}

// -------------------- Game loop --------------------

const TargetTPS = 120.0

var GameInstance *Game

func (g *Game) Update() error {
	dt := float32(1.0 / TargetTPS)
	g.Player.Update(dt)
	return nil
}

func (g *Game) drawScene(dst *ebiten.Image) {
	// background
	ebitenutil.DrawRect(dst, 0, 0, float64(g.ScreenWidth), float64(g.ScreenHeight),
		color.RGBA{R: 0, G: 100, B: 200, A: 255})

	// player (16x16 square)
	const w = 16.0

	frame := heroAnimationManager.GetCurrentFrame()
	op := &ebiten.DrawImageOptions{}
	// figure out how to scale it to 64
	tgtWidth := float64(64)
	l := frame.Bounds().Dx()
	s := float64(tgtWidth) / float64(l)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(-tgtWidth/2, -tgtWidth/2)
	op.GeoM.Translate(float64(g.Player.Pos.X), float64(g.Player.Pos.Y))
	dst.DrawImage(frame, op)
	// particles
	for _, w := range g.Player.Weapons {
		w.ParticleEmitter.Draw(dst)
	}

	// Render the hearts
	// op := &ebiten.DrawImageOptions{}
	// op.GeoM.Translate(float64(i*32), 0)
	// dst.DrawImage(frame, op)

	for i, frame := range statusBarAnimationManager.GetHeartFrames() {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(5+float64(i*35), float64(g.ScreenHeight)-32-64)
		dst.DrawImage(frame, op)
	}

	for i, frame := range statusBarAnimationManager.GetManaFrames() {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(5+float64(i*35), float64(g.ScreenHeight)-32-16)
		dst.DrawImage(frame, op)
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
		"PixelSize":    float32(1),               // 1..4
		"Vignette":     float32(0.4),             // 0..1
		"Grain":        float32(0.07),            // 0..0.4
		"Bloom":        float32(0.55),            // 0..1
		"Aberration":   float32(0.0003),          // 0..0.005
		"Saturation":   float32(1.1),             // 0.8..1.3
		"Contrast":     float32(1.2),             // 0.9..1.2
		"Gamma":        float32(1.2),             // 0.9..1.4
		"Border":       float32(1.5),             // intensity (try 0.8–1.5)
		"BorderClamp":  float32(.3),              // max darken (0.15–0.35)
		"BorderRadius": float32(1),               // neighbor distance in px (1–2)
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

func StartGame() {
	Init()
	const (
		logicalW = 320 * 4
		logicalH = 240 * 4
		scale    = 1
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

	_, _ = smokeWeapon, earthWeapon // silence unused

	player := Player{
		Pos:               &Vec2{X: 100, Y: 100},
		Direction:         Vec2Zero,
		Speed:             80, // px/sec
		Weapons:           []Weapon{fireWeapon},
		MaxHealth:         5,
		Health:            5,
		MaxMana:           2,
		Mana:              2,
		ManaRegenRate:     .2, // mana per second
		ManaRegenCooldown: 0,
		StrifeDuration:    1.5,             // length
		StrifeCooldown:    time.Second * 1, // cooldown
		StrifeMultiplier:  2.5,             // speed multiplier
		StrifeDecay:       2,               // decay rate
		LastStrife:        time.Now(),
		StrifeTime:        0, // current time left in strife

	}

	// -- Set up animators --
	heroAnimationManager = util.NewCharacterWalkAnimator(heroImagePath)
	statusBarAnimationManager = util.NewStatusBarAnimationManager("assets/toolbar/health.png", "assets/toolbar/mana.png", player.MaxHealth, player.MaxMana)

	statusBarAnimationManager.DecrementHeart(900, "health")
	statusBarAnimationManager.IncrementHeart(10, "health")

	game := &Game{
		ScreenWidth:  logicalW,
		ScreenHeight: logicalH,
		Player:       player,
		startedAt:    time.Now(),
	}

	GameInstance = game

	sh, err := ebiten.NewShader([]byte(retroShaderSrc))
	if err != nil {
		log.Fatal(err)
	}
	game.retroShader = sh

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
