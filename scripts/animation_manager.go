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
	heartDFAs []*DFA
	// number of mana
	manaDFAs []*DFA
}

/*
Load the sprite sheet.
Assume it is 4x9 in order, and each frame is 8x8 pixels.
*/
func loadDFA(spritSheetPath string, row int, startCol int, numCols int, width int) *DFA {
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
		curState.AddTransition("next", start)
		col++
	}
	return &DFA{
		startState:   start,
		currentState: start,
	}
}

func (dfa *DFA) FullyConnectToOther(dfa2 *DFA, input string) {
	// Allows one dfa to transition to another
	var start *state = dfa.startState
	for start != nil {
		start.AddTransition(input, dfa2.startState)
		start = start.transitions["next"]
		if start == dfa.startState {
			break
		}
	}
}

func NewCharacterWalkingAnimator(spriteSheet string) *WalkingAnimationManager {
	upDFA := loadDFA(spriteSheet, 8, 0, 9, 64)
	leftDFA := loadDFA(spriteSheet, 9, 0, 9, 64)
	downDFA := loadDFA(spriteSheet, 10, 0, 9, 64)
	rightDFA := loadDFA(spriteSheet, 11, 0, 9, 64)

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

	strifeLeftDFA := loadDFA(spriteSheet, 9, 1, 1, 64)
	strifeRightDFA := loadDFA(spriteSheet, 11, 1, 1, 64)
	strifeUpDFA := loadDFA(spriteSheet, 8, 3, 1, 64)
	strifeDownDFA := loadDFA(spriteSheet, 10, 3, 1, 64)

	// connect walk left to strife left
	leftDFA.FullyConnectToOther(strifeLeftDFA, "strife")
	rightDFA.FullyConnectToOther(strifeRightDFA, "strife")
	downDFA.FullyConnectToOther(strifeDownDFA, "strife")
	upDFA.FullyConnectToOther(strifeUpDFA, "strife")

	blockUpDFA := loadDFA(spriteSheet, 4, 4, 1, 64)
	blockLeftDFA := loadDFA(spriteSheet, 5, 5, 1, 64)
	blockDownDFA := loadDFA(spriteSheet, 6, 6, 1, 64)
	blockRightDFA := loadDFA(spriteSheet, 7, 7, 1, 64)

	// connect up walk to up block on "block" input
	upDFA.FullyConnectToOther(blockUpDFA, "block")
	leftDFA.FullyConnectToOther(blockLeftDFA, "block")
	downDFA.FullyConnectToOther(blockDownDFA, "block")
	rightDFA.FullyConnectToOther(blockRightDFA, "block")

	return &WalkingAnimationManager{
		curState: downDFA.startState,
	}
}

func NewStatusBarAnimationManager(heartSpriteSheet string, manaSpriteSheet string, numHearts rune, numMana rune) *StatusBarAnimationManager {
	heartDFAs := make([]*DFA, 0, numHearts)
	for i := 0; i < int(numHearts); i++ {
		heartDFAs = append(heartDFAs, loadDFA(heartSpriteSheet, 0, 0, 5, 32))
	}

	manaDFAs := make([]*DFA, 0, numMana)
	for i := 0; i < int(numMana); i++ {
		manaDFAs = append(manaDFAs, loadDFA(manaSpriteSheet, 0, 0, 5, 32))
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

		if dfa.HasNextState("next") {
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

		if dfa.HasNextState("prev") {
			dfa.currentState = dfa.NextState("prev")
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

		if dfa.HasNextState("prev") {
			dfa.currentState = dfa.NextState("prev")
			amount--
		} else {
			// already depleted
			dfaIndex--
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

	if len(overrideInput) > 0 {
		// set override state if exists
		override := am.curState.transitions[overrideInput]
		if override != nil {
			am.overrideState = override
		}
	} else {
		am.overrideState = nil
	}

	if am.overrideState != nil {
		am.overrideState = am.overrideState.transitions["next"]
		return
	}

	am.timeInState += dt

	if am.timeInState < 150*time.Millisecond && len(overrideInput) == 0 {
		return
	}

	// not moving = freeze unless override
	if !moving && len(overrideInput) == 0 {
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
		nextState = am.curState.transitions["next"]
	}

	am.curState = nextState
}
