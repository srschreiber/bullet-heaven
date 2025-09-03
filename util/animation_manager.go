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

type StatusBarAnimationManager struct {
	// number of hearts
	heartDFAs []*DFA
	// number of mana
	manaDFAs []*DFA
}

/*
Load the sprite sheet.
Assume it is 4x9 in order, and each frame is 8x8 pixels.
*/
func loadDFA(spritSheetPath string, row int, numCols int, width int, nextInput string, previousInput string) *DFA {
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
			curState.AddTransition(nextInput, nextState)
			// add transition back to where came from
			nextState.AddTransition(previousInput, curState)
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

func NewCharacterWalkAnimator(spriteSheet string) *AnimationManager {
	upDFA := loadDFA(spriteSheet, 0, 9, 64, "step", "backstep")
	leftDFA := loadDFA(spriteSheet, 1, 9, 64, "step", "backstep")
	downDFA := loadDFA(spriteSheet, 2, 9, 64, "step", "backstep")
	rightDFA := loadDFA(spriteSheet, 3, 9, 64, "step", "backstep")

	return &AnimationManager{
		walkingUpDFA:    upDFA,
		walkingLeftDFA:  leftDFA,
		walkingDownDFA:  downDFA,
		walkingRightDFA: rightDFA,
		curDFA:          rightDFA,
	}
}

func NewStatusBarAnimationManager(heartSpriteSheet string, manaSpriteSheet string, numHearts int, numMana int) *StatusBarAnimationManager {
	heartDFAs := make([]*DFA, 0, numHearts)
	for i := 0; i < numHearts; i++ {
		heartDFAs = append(heartDFAs, loadDFA(heartSpriteSheet, i, 0, 32, "reduce", "increase"))
	}

	manaDFAs := make([]*DFA, 0, numMana)
	for i := 0; i < numMana; i++ {
		manaDFAs = append(manaDFAs, loadDFA(manaSpriteSheet, i, 0, 32, "reduce", "increase"))
		// one for increase
		manaDFAs = append(manaDFAs, loadDFA(manaSpriteSheet, i, 0, 32, "red", "increase"))
	}

	return &StatusBarAnimationManager{
		heartDFAs: heartDFAs,
		manaDFAs:  manaDFAs,
	}
}

func (sbam *StatusBarAnimationManager) GetHeartFrames() []*ebiten.Image {
	frames := make([]*ebiten.Image, 0, len(sbam.heartDFAs))
	for _, dfa := range sbam.heartDFAs {
		frames = append(frames, dfa.currentState.stateData.(*ebiten.Image))
	}
	return frames
}

func (sbam *StatusBarAnimationManager) GetManaFrames() []*ebiten.Image {
	frames := make([]*ebiten.Image, 0, len(sbam.manaDFAs))
	for _, dfa := range sbam.manaDFAs {
		frames = append(frames, dfa.currentState.stateData.(*ebiten.Image))
	}
	return frames
}

func (sbam *StatusBarAnimationManager) DecrementHeart(amount int, t string) {
	dfaIndex := 0
	var dfas []*DFA

	if t == "health" {
		dfas = sbam.heartDFAs
	} else if t == "mana" {
		dfas = sbam.manaDFAs
	}

	for dfaIndex < len(dfas) && amount > 0 {
		dfa := dfas[dfaIndex]

		if dfa.HasNextState("reduce") {
			dfa.currentState = dfa.NextState("reduce")
			amount--
		} else {
			// already depleted
			dfaIndex++
		}
	}
}

func (sbam *StatusBarAnimationManager) IncrementHeart(amount int, t string) {
	dfaIndex := len(sbam.heartDFAs) - 1
	var dfas []*DFA

	if t == "health" {
		dfas = sbam.heartDFAs
	} else if t == "mana" {
		dfas = sbam.manaDFAs
	}

	for dfaIndex >= 0 && amount > 0 {
		dfa := dfas[dfaIndex]

		if dfa.HasNextState("increase") {
			dfa.currentState = dfa.NextState("increase")
			amount--
		} else {
			// already depleted
			dfaIndex--
		}
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
