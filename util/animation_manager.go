/*
Helps manage animations.

Given:
1. A sprite sheet that has idle, walk left left foot forward, walk left both feet together, walk left right foot forward, etc
*/
package util

import (
	"time"

	"github.com/hajimehoshi/ebiten"
)

// Define all animation states

type AnimationState int

const (
	Idle AnimationState = iota
	WalkLeftLegForward
	WalkLeftBothFeetTogether
	WalkLeftRightLegForward
	WalkRightLegForward
	WalkRightFeetTogether
	WalkDownLegForward
)

type AnimationManager struct {
	// sprite sheet for the animations
	spriteSheet *ebiten.Image
	// current animation state
	currentState  AnimationState
	previousState AnimationState
	// animation frames
	currentSprite *ebiten.Image
	// frame duration
	frameDuration time.Duration
	timeInState   time.Duration
}
