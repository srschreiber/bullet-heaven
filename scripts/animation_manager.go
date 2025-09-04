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
	walkingLeftDFA  *DFA
	walkingRightDFA *DFA
	walkingUpDFA    *DFA
	walkingDownDFA  *DFA
	// last dfa for when no input is received, or detecting when just arrived at new dfa (reset to start)
	curDFA         *DFA
	timeInState    time.Duration
	strifeLeftDFA  *DFA
	strifeRightDFA *DFA
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
func loadDFA(spritSheetPath string, row int, startCol int, numCols int, width int, nextInput string, previousInput string) *DFA {
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
			prevState.AddTransition(nextInput, curState)
			curState.AddTransition(previousInput, prevState)
		}

		if start == nil {
			start = curState
		}

		prevState = curState
		col++
	}
	return &DFA{
		startState:   start,
		currentState: start,
	}
}

func NewCharacterWalkingAnimator(spriteSheet string, startRow int) *WalkingAnimationManager {
	upDFA := loadDFA(spriteSheet, startRow, 0, 9, 64, "step", "backstep")
	leftDFA := loadDFA(spriteSheet, startRow+1, 0, 9, 64, "step", "backstep")
	downDFA := loadDFA(spriteSheet, startRow+2, 0, 9, 64, "step", "backstep")
	rightDFA := loadDFA(spriteSheet, startRow+3, 0, 9, 64, "step", "backstep")

	strifeLeftDFA := loadDFA(spriteSheet, startRow+1, 1, 1, 64, "step", "backstep")
	strifeRightDFA := loadDFA(spriteSheet, startRow+3, 1, 1, 64, "step", "backstep")

	return &WalkingAnimationManager{
		walkingUpDFA:    upDFA,
		walkingLeftDFA:  leftDFA,
		walkingDownDFA:  downDFA,
		walkingRightDFA: rightDFA,
		curDFA:          downDFA,
		strifeLeftDFA:   strifeLeftDFA,
		strifeRightDFA:  strifeRightDFA,
	}
}

func NewStatusBarAnimationManager(heartSpriteSheet string, manaSpriteSheet string, numHearts rune, numMana rune) *StatusBarAnimationManager {
	heartDFAs := make([]*DFA, 0, numHearts)
	for i := 0; i < int(numHearts); i++ {
		heartDFAs = append(heartDFAs, loadDFA(heartSpriteSheet, 0, 0, 5, 32, "reduce", "increase"))
	}

	manaDFAs := make([]*DFA, 0, numMana)
	for i := 0; i < int(numMana); i++ {
		manaDFAs = append(manaDFAs, loadDFA(manaSpriteSheet, 0, 0, 5, 32, "reduce", "increase"))
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

func (sbam *StatusBarAnimationManager) HasHearts(t string) bool {
	dfaIndex := 0
	var dfas []*DFA

	if t == "health" {
		dfas = sbam.heartDFAs
	}

	if t == "mana" {
		dfas = sbam.manaDFAs
	}

	for dfaIndex < len(dfas) {
		dfa := dfas[dfaIndex]

		if dfa.HasNextState("reduce") {
			return true
		}
		dfaIndex++
	}
	return false
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
	var dfas []*DFA

	if t == "health" {
		dfas = sbam.heartDFAs
	} else if t == "mana" {
		dfas = sbam.manaDFAs
	}
	dfaIndex := len(dfas) - 1
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

func (am *WalkingAnimationManager) GetCurrentFrame() *ebiten.Image {
	if am.curDFA != nil {
		return am.curDFA.currentState.stateData.(*ebiten.Image)
	}
	return nil
}

func (am *WalkingAnimationManager) UpdateByDirection(dirX, dirY float64, dt time.Duration, strife bool, moving bool) {
	var nextDFA *DFA
	am.timeInState += dt

	if am.timeInState < 150*time.Millisecond && !strife {
		return
	}

	am.timeInState = 0

	if !strife {
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
	} else {
		if dirX > 0 {
			nextDFA = am.strifeRightDFA
		} else {
			nextDFA = am.strifeLeftDFA
		}
	}

	if nextDFA == am.curDFA && moving {
		am.curDFA.currentState = am.curDFA.NextState("step")
	} else {
		am.curDFA = nextDFA
		// reset to beginning
		am.curDFA.currentState = am.curDFA.startState
	}
}
