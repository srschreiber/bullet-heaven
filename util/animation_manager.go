package util

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

type AnimationManager struct {
	walkingLeftDFA  *DFA
	walkingRightDFA *DFA
	walkingUpDFA    *DFA
	walkingDownDFA  *DFA
	// last dfa for when no input is received, or detecting when just arrived at new dfa (reset to start)
	curDFA      *DFA
	timeInState time.Duration
}

/*
Load the sprite sheet.
Assume it is 4x9 in order, and each frame is 8x8 pixels.
*/
func loadDFA(spritSheetPath string, row int, numCols int, width int, input string) *DFA {
	col := 0
	var start *state
	var curState *state
	for col < numCols {
		rect := image.Rect(col*width, row*width, (col+1)*width, (row+1)*width)
		spriteSheet, _, err := ebitenutil.NewImageFromFile(spritSheetPath)
		if err != nil {
			log.Fatal(err)
		}
		frame := spriteSheet.SubImage(rect).(*ebiten.Image)
		nextState := NewState("frame"+strconv.Itoa(col), frame)
		if curState != nil {
			curState.AddTransition(input, nextState)
		}
		curState = nextState
		if start == nil {
			start = curState
		}
		col++
	}
	return &DFA{
		startState:   start,
		currentState: start,
	}
}

func NewAnimationManager(spriteSheet string) *AnimationManager {
	upDFA := loadDFA(spriteSheet, 0, 9, 64, "step")
	leftDFA := loadDFA(spriteSheet, 1, 9, 64, "step")
	downDFA := loadDFA(spriteSheet, 2, 9, 64, "step")
	rightDFA := loadDFA(spriteSheet, 3, 9, 64, "step")

	return &AnimationManager{
		walkingUpDFA:    upDFA,
		walkingLeftDFA:  leftDFA,
		walkingDownDFA:  downDFA,
		walkingRightDFA: rightDFA,
		curDFA:          rightDFA,
	}
}

func (am *AnimationManager) GetCurrentFrame() *ebiten.Image {
	if am.curDFA != nil {
		return am.curDFA.currentState.stateData.(*ebiten.Image)
	}
	return nil
}

func (am *AnimationManager) UpdateByDirection(dirX, dirY float64, dt time.Duration) {
	var nextDFA *DFA
	am.timeInState += dt

	if am.timeInState < 150*time.Millisecond {
		return
	}

	am.timeInState = 0

	if dirX == 0 && dirY == 0 {
		// If no direction is given, reset to idle animation
		nextDFA = am.walkingDownDFA
		am.curDFA = nextDFA
		return
	} else {
		// find direction it is most in
		if math.Abs(dirX) > math.Abs(dirY) {
			if dirX > 0 {
				nextDFA = am.walkingRightDFA
			} else {
				nextDFA = am.walkingLeftDFA
			}
		} else {
			if dirY > 0 {
				nextDFA = am.walkingDownDFA
			} else {
				nextDFA = am.walkingUpDFA
			}
		}
	}

	if nextDFA == am.curDFA {
		am.curDFA.currentState = am.curDFA.NextState("step")
	} else {
		am.curDFA = nextDFA
		// reset to beginning
		am.curDFA.currentState = am.curDFA.startState
	}
}
