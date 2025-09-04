package scripts

import (
	"image"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type AnimationState int

type WalkingAnimationManager struct {
	curState      *state
	overrideState *state
	timeInState   time.Duration
}

type StatusBarAnimationManager struct {
	// number of hearts
	heartStates []*state
	// number of mana
	manaStates []*state
}

/*
Load the sprite sheet.
Assume it is 4x9 in order, and each frame is 8x8 pixels.
*/
func loadDFA(spritSheetPath string, row int, startCol int, numCols int, width int, loop bool) *state {
	col := startCol
	var start *state
	var prevState *state
	for col-startCol < numCols {
		rect := image.Rect(col*width, row*width, (col+1)*width, (row+1)*width)
		spriteSheet, _, err := ebitenutil.NewImageFromFile(spritSheetPath)
		if err != nil {
			log.Fatal(err)
		}
		frame := spriteSheet.SubImage(rect).(*ebiten.Image)
		curState := NewState("frame"+strconv.Itoa(col), frame)

		if prevState != nil {
			prevState.AddTransition("next", curState)
			curState.AddTransition("prev", prevState)
		}

		if start == nil {
			start = curState
		}

		prevState = curState
		// connect last state to start for loop (if not done, will be overwritten)
		if loop {
			curState.AddTransition("next", start)
		} else {
			curState.AddTransition("next", curState)
		}
		col++
	}
	start.AddTransition("prev", start)

	return start
}

func (s *state) FullyConnectToOther(state2 *state, input string) {
	// Allows one dfa to transition to another
	var it *state = s
	for it != nil {
		prev := it
		it.AddTransition(input, state2)
		it = it.transitions["next"]
		if it == s || prev == it {
			break
		}
	}
}

func NewCharacterWalkingAnimator(spriteSheet string) *WalkingAnimationManager {
	upDFA := loadDFA(spriteSheet, 8, 0, 9, 64, true)
	leftDFA := loadDFA(spriteSheet, 9, 0, 9, 64, true)
	downDFA := loadDFA(spriteSheet, 10, 0, 9, 64, true)
	rightDFA := loadDFA(spriteSheet, 11, 0, 9, 64, true)

	leftDFA.FullyConnectToOther(upDFA, "up")
	downDFA.FullyConnectToOther(upDFA, "up")
	rightDFA.FullyConnectToOther(upDFA, "up")
	upDFA.FullyConnectToOther(leftDFA, "left")
	downDFA.FullyConnectToOther(leftDFA, "left")
	rightDFA.FullyConnectToOther(leftDFA, "left")
	upDFA.FullyConnectToOther(downDFA, "down")
	leftDFA.FullyConnectToOther(downDFA, "down")
	rightDFA.FullyConnectToOther(downDFA, "down")
	upDFA.FullyConnectToOther(rightDFA, "right")
	leftDFA.FullyConnectToOther(rightDFA, "right")
	downDFA.FullyConnectToOther(rightDFA, "right")

	strifeLeftDFA := loadDFA(spriteSheet, 9, 1, 1, 64, false)
	strifeRightDFA := loadDFA(spriteSheet, 11, 1, 1, 64, false)
	strifeUpDFA := loadDFA(spriteSheet, 8, 3, 1, 64, false)
	strifeDownDFA := loadDFA(spriteSheet, 10, 3, 1, 64, false)

	// connect walk left to strife left
	leftDFA.FullyConnectToOther(strifeLeftDFA, "strife")
	rightDFA.FullyConnectToOther(strifeRightDFA, "strife")
	downDFA.FullyConnectToOther(strifeDownDFA, "strife")
	upDFA.FullyConnectToOther(strifeUpDFA, "strife")

	blockUpDFA := loadDFA(spriteSheet, 4, 0, 8, 64, false)
	blockLeftDFA := loadDFA(spriteSheet, 5, 0, 8, 64, false)
	blockDownDFA := loadDFA(spriteSheet, 6, 0, 8, 64, false)
	blockRightDFA := loadDFA(spriteSheet, 7, 0, 8, 64, false)

	// connect up walk to up block on "block" input
	upDFA.FullyConnectToOther(blockUpDFA, "block")
	leftDFA.FullyConnectToOther(blockLeftDFA, "block")
	downDFA.FullyConnectToOther(blockDownDFA, "block")
	rightDFA.FullyConnectToOther(blockRightDFA, "block")

	return &WalkingAnimationManager{
		curState: downDFA,
	}
}

func NewStatusBarAnimationManager(heartSpriteSheet string, manaSpriteSheet string, numHearts rune, numMana rune) *StatusBarAnimationManager {
	heartStates := make([]*state, 0, numHearts)
	for i := 0; i < int(numHearts); i++ {
		heartStates = append(heartStates, loadDFA(heartSpriteSheet, 0, 0, 5, 32, false))
	}

	manaStates := make([]*state, 0, numMana)
	for i := 0; i < int(numMana); i++ {
		manaStates = append(manaStates, loadDFA(manaSpriteSheet, 0, 0, 5, 32, false))
	}

	return &StatusBarAnimationManager{
		heartStates: heartStates,
		manaStates:  manaStates,
	}
}

func (sbam *StatusBarAnimationManager) GetHeartFrames() []*ebiten.Image {
	frames := make([]*ebiten.Image, 0, len(sbam.heartStates))
	for _, state := range sbam.heartStates {
		frames = append(frames, state.stateData.(*ebiten.Image))
	}
	return frames
}

func (sbam *StatusBarAnimationManager) GetManaFrames() []*ebiten.Image {
	frames := make([]*ebiten.Image, 0, len(sbam.manaStates))
	for _, state := range sbam.manaStates {
		frames = append(frames, state.stateData.(*ebiten.Image))
	}
	return frames
}

func (sbam *StatusBarAnimationManager) HasHearts(t string) bool {
	statesIndex := 0
	var states []*state

	if t == "health" {
		states = sbam.heartStates
	}

	if t == "mana" {
		states = sbam.manaStates
	}

	for statesIndex < len(states) {
		state := states[statesIndex]

		if state.transitions["next"] != state {
			return true
		}
		statesIndex++
	}
	return false
}

func (sbam *StatusBarAnimationManager) DecrementHeart(amount int, t string) {
	stateIndex := 0
	var states []*state

	if t == "health" {
		states = sbam.heartStates
	} else if t == "mana" {
		states = sbam.manaStates
	}

	for stateIndex < len(states) && amount > 0 {
		state := states[stateIndex]

		if state.transitions["next"] != state {
			states[stateIndex] = state.transitions["next"]
			amount--
		} else {
			// already depleted
			stateIndex++
		}
	}
}

func (sbam *StatusBarAnimationManager) IncrementHeart(amount int, t string) {
	var states []*state

	if t == "health" {
		states = sbam.heartStates
	} else if t == "mana" {
		states = sbam.manaStates
	}
	stateIndex := len(states) - 1
	for stateIndex >= 0 && amount > 0 {
		state := states[stateIndex]

		if state.transitions["prev"] != state {
			states[stateIndex] = state.transitions["prev"]
			amount--
		} else {
			// already depleted
			stateIndex--
		}
	}
}

func (am *WalkingAnimationManager) GetCurrentFrame() *ebiten.Image {
	if am.overrideState != nil {
		return am.overrideState.stateData.(*ebiten.Image)
	}

	if am.curState != nil {
		return am.curState.stateData.(*ebiten.Image)
	}
	return nil
}

func (am *WalkingAnimationManager) UpdateByDirection(dirX, dirY float64, dt time.Duration, moving bool, overrideInput string) {
	var nextState *state
	am.timeInState += dt

	if len(overrideInput) > 0 {
		// Init override if not already set
		if am.overrideState == nil {
			override := am.curState.transitions[overrideInput]
			if override != nil {
				am.overrideState = override
			}
		}
	} else {
		am.overrideState = nil
	}

	if am.overrideState != nil {
		am.overrideState = am.overrideState.transitions["next"]
		return
	}

	if am.timeInState < 150*time.Millisecond && len(overrideInput) == 0 {
		return
	}

	am.timeInState = 0

	dirInput := ""
	// find direction it is most in
	if math.Abs(dirX) > math.Abs(dirY/2) {
		if dirX > 0 {
			dirInput = "right"
		} else {
			dirInput = "left"
		}
	} else {
		if dirY > 0 {
			dirInput = "down"
		} else {
			dirInput = "up"
		}
	}

	nextState = am.curState.transitions[dirInput]
	if nextState == nil {
		// nil means within same dfa
		if !moving {
			return
		}
		nextState = am.curState.transitions["next"]
	}

	am.curState = nextState
}
