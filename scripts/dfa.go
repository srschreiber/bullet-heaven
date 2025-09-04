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

func (s *state) AddNext(nextState *state) *state {
	s.AddTransition("next", nextState)
	return s
}

func (s *state) AddPrev(prevState *state) *state {
	s.AddTransition("prev", prevState)
	return s
}

func (s *state) SendInput(input string) *state {
	nextState, exists := s.transitions[input]
	if exists {
		return nextState
	}
	return nil
}

func (s *state) Next() *state {
	return s.SendInput("next")
}

func (s *state) Prev() *state {
	return s.SendInput("prev")
}
