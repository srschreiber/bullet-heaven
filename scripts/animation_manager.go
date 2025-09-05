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
			prevState.AddNext(curState)
			curState.AddPrev(prevState)
		}

		if start == nil {
			start = curState
		}

		prevState = curState
		// connect last state to start for loop (if not done, will be overwritten)
		if loop {
			curState.AddNext(start)
		} else {
			curState.AddNext(curState) // stay on last frame
		}
		col++
	}
	start.AddPrev(start)

	return start
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
			override := am.curState.SendInput(overrideInput)
			if override != nil {
				am.overrideState = override
			}
		}
	} else {
		am.overrideState = nil
	}

	if am.overrideState != nil {
		am.overrideState = am.overrideState.Next()
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

	nextState = am.curState.SendInput(dirInput)
	if nextState == nil {
		// nil means within same dfa
		if !moving {
			return
		}
		nextState = am.curState.Next()
	}

	am.curState = nextState
}
