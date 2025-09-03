package util

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type AnimationState int

const (
	// Left-walk cycle frames
	WalkLeftLegForward       AnimationState = iota // phase 0
	WalkLeftBothFeetTogether                       // phase 1
	WalkLeftRightLegForward                        // phase 2

	// Right-walk cycle frames
	WalkRightLegForward
	WalkRightFeetTogether
	WalkRightLeftLegForward
)

// AnimationManager drives a reliable L-Both-R-Both cadence.
// We keep a small 4-step wheel [0,1,2,1] and map it to L/R states.
type AnimationManager struct {
	spriteSheet *ebiten.Image
	frames      map[AnimationState]image.Rectangle

	// phase wheel index 0..3 over [0,1,2,1]
	phaseIdx int
	// cached phase value derived from phaseIdx (0,1,2)
	phase int

	// facingLeft determines which branch to use for mapping
	facingLeft bool

	frameDuration time.Duration
	timeInPhase   time.Duration
}

// NewAnimationManager initializes paused on "feet together" facing right.
func NewAnimationManager(spriteSheet *ebiten.Image, frames map[AnimationState]image.Rectangle, frameDuration time.Duration) *AnimationManager {
	am := &AnimationManager{
		spriteSheet:   spriteSheet,
		frames:        frames,
		frameDuration: frameDuration,
	}
	am.phaseIdx = 1 // [0,1,2,1] -> index 1 is "both"
	am.phase = 1
	am.facingLeft = false
	return am
}

// CurrentFrame returns the subimage for the current state.
func (am *AnimationManager) CurrentFrame() *ebiten.Image {
	rect := am.frames[am.currentState()]
	return am.spriteSheet.SubImage(rect).(*ebiten.Image)
}

// UpdateByDirection advances animation based on direction and dt.
// Rules (as requested):
// - If mostly horizontal: animate left/right normally.
// - If mostly vertical but ANY horizontal exists: animate using that left/right sign.
// - If perfectly vertical (dirX == 0): animate with current facing (don't flip).
// - If stopped: pause on feet-together for last facing.
func (am *AnimationManager) UpdateByDirection(dirX, dirY float64, dt time.Duration) {
	am.timeInPhase += dt

	// Determine target facing from direction
	ax, ay := abs(dirX), abs(dirY)
	moving := (ax > 0) || (ay > 0)

	if !moving {
		// Pause on Both in current facing
		am.snapToBoth()
		return
	}

	switch {
	case ax >= ay:
		// Horizontal dominates → face by sign
		am.facingLeft = dirX < 0
	case dirX != 0:
		// Vertical dominates but we have slight horizontal → use its sign
		am.facingLeft = dirX < 0
		// else perfectly vertical: keep current facing (no change)
	}

	// Advance cadence on timer
	if am.timeInPhase >= am.frameDuration {
		am.timeInPhase = 0
		am.phaseIdx = (am.phaseIdx + 1) % 4 // 0,1,2,1,0,1...
		am.phase = wheelPhase(am.phaseIdx)  // map idx -> {0,1,2}
	}
}

// --- internals ---

// Map wheel index to phase value {0,1,2,1}
func wheelPhase(idx int) int {
	switch idx & 3 {
	case 0:
		return 0
	case 1:
		return 1
	case 2:
		return 2
	default:
		return 1
	}
}

// currentState maps (phase, facingLeft) → concrete sprite frame.
func (am *AnimationManager) currentState() AnimationState {
	if am.facingLeft {
		switch am.phase {
		case 0:
			return WalkLeftLegForward
		case 1:
			return WalkLeftBothFeetTogether
		default: // 2
			return WalkLeftRightLegForward
		}
	}
	// facing right
	switch am.phase {
	case 0:
		return WalkRightLegForward
	case 1:
		return WalkRightFeetTogether
	default: // 2
		return WalkRightLeftLegForward
	}
}

// snapToBoth pauses on the "feet together" frame for the current facing.
func (am *AnimationManager) snapToBoth() {
	am.phaseIdx = 1 // index 1 in [0,1,2,1] is the "both" phase
	am.phase = 1
	am.timeInPhase = 0
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
