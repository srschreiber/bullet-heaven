package scripts

type state struct {
	// Define the fields for the DFA state
	id          string
	transitions map[string]*state
	// stateData useful for caching information at a state that could be useful
	// such as the current animation frame
	stateData interface{}
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
