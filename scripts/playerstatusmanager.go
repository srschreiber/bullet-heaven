package scripts

import "github.com/hajimehoshi/ebiten/v2"

type StatusBarEnum int

const (
	HealthStatus StatusBarEnum = iota
	ManaStatus
	StaminaStatus
)

type StatusBarAnimationManager struct {
	// number of hearts
	heartStates []*state
	// number of mana
	manaStates    []*state
	staminaStates []*state
}

func NewStatusBarAnimationManager(heartSpriteSheet string, manaSpriteSheet string, staminaSpriteSheet string, numHearts rune, numMana rune, numStamina rune) *StatusBarAnimationManager {
	heartStates := make([]*state, 0, numHearts)
	for i := 0; i < int(numHearts); i++ {
		heartStates = append(heartStates, loadDFA(heartSpriteSheet, 0, 0, 5, 32, false))
	}

	manaStates := make([]*state, 0, numMana)
	for i := 0; i < int(numMana); i++ {
		manaStates = append(manaStates, loadDFA(manaSpriteSheet, 0, 0, 5, 32, false))
	}

	staminaStates := make([]*state, 0, numStamina)
	for i := 0; i < int(numStamina); i++ {
		staminaStates = append(staminaStates, loadDFA(staminaSpriteSheet, 0, 0, 5, 32, false))
	}

	return &StatusBarAnimationManager{
		heartStates:   heartStates,
		manaStates:    manaStates,
		staminaStates: staminaStates,
	}
}

func (sbam *StatusBarAnimationManager) GetStatusFrames(status StatusBarEnum) []*ebiten.Image {
	var states []*state = sbam.GetStates(status)
	frames := make([]*ebiten.Image, 0, len(states))
	for _, state := range states {
		frames = append(frames, state.stateData.(*ebiten.Image))
	}
	return frames
}

func (sbam *StatusBarAnimationManager) HasHearts(t StatusBarEnum) bool {
	statesIndex := 0
	var states []*state = sbam.GetStates(t)

	for statesIndex < len(states) {
		state := states[statesIndex]

		if state.Next() != state {
			return true
		}
		statesIndex++
	}
	return false
}

func (sbam *StatusBarAnimationManager) GetStates(t StatusBarEnum) []*state {
	switch t {
	case HealthStatus:
		return sbam.heartStates
	case ManaStatus:
		return sbam.manaStates
	case StaminaStatus:
		return sbam.staminaStates
	}
	return nil
}

func (sbam *StatusBarAnimationManager) DecrementHeart(amount int, t StatusBarEnum) {
	stateIndex := 0
	var states []*state = sbam.GetStates(t)

	for stateIndex < len(states) && amount > 0 {
		state := states[stateIndex]

		if state.Next() != state {
			states[stateIndex] = state.Next()
			amount--
		} else {
			// already depleted
			stateIndex++
		}
	}
}

func (sbam *StatusBarAnimationManager) IncrementHeart(amount int, t StatusBarEnum) {
	var states []*state = sbam.GetStates(t)

	stateIndex := len(states) - 1
	for stateIndex >= 0 && amount > 0 {
		state := states[stateIndex]

		if state.Prev() != state {
			states[stateIndex] = state.Prev()
			amount--
		} else {
			// already depleted
			stateIndex--
		}
	}
}
