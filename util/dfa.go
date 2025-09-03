package util

type state struct {
	// Define the fields for the DFA state
	id          string
	transitions map[string]*state
	// stateData useful for caching information at a state that could be useful
	// such as the current animation frame
	stateData interface{}
}

type DFA struct {
	startState   *state
	currentState *state
}

func NewState(id string, stateData interface{}) *state {
	newState := &state{
		id:          id,
		stateData:   stateData,
		transitions: make(map[string]*state),
	}
	return newState
}

func (s *state) AddTransition(input string, nextState *state) *state {
	s.transitions[input] = nextState
	return s.transitions[input]
}

func (dfa *DFA) HasNextState(input string) bool {
	if dfa.currentState != nil {
		if _, ok := dfa.currentState.transitions[input]; ok {
			return true
		}
	}
	return false
}

func (dfa *DFA) NextState(input string) *state {
	if dfa.currentState != nil {
		if nextState, ok := dfa.currentState.transitions[input]; ok {
			dfa.currentState = nextState
		} else {
			// reset
			dfa.currentState = dfa.startState
		}
	}
	return dfa.currentState
}
