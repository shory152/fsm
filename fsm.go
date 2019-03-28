package fsm

// FSM's state
type State int

// FSM's input event
type Event int

// Action associated to a state
type Action interface {
	Do()
}

// Action func associated to a state
type ActionFunc func()

func (f ActionFunc) Do() {
	f()
}

// export ConfigState for configure each state of StateMachine
type ConfigState interface {
	Accept(e Event, next State) ConfigState
	OnEnter(a Action) ConfigState
	OnEnterFrom(prev State, a Action) ConfigState
	OnExit(a Action) ConfigState
}

type interState struct {
	id          State
	enterAction Action
	enterFrom   map[State]Action
	exitAction  Action
	next        map[Event]*interState
	fsm         *stateMachine
}

// this state accept e, then transfer to nextS
func (is *interState) Accept(e Event, nextS State) ConfigState {
	nis := is.fsm.ConfigState(nextS)
	if is.next == nil {
		is.next = make(map[Event]*interState)
	}
	is.next[e] = nis.(*interState)
	return is
}

// execute act when enter this state
func (is *interState) OnEnter(act Action) ConfigState {
	is.enterAction = act
	return is
}

// execute act when enter this state from prev
func (is *interState) OnEnterFrom(prev State, act Action) ConfigState {
	if is.enterFrom == nil {
		is.enterFrom = make(map[State]Action)
	}
	is.enterFrom[prev] = act
	return is
}

// execute act when exit this state
func (is *interState) OnExit(act Action) ConfigState {
	is.exitAction = act
	return is
}

// fsm which be driven step-by-step
type StepFSM interface {
	ConfigState(State) ConfigState
	Step(Event)
	Close()
}

// fsm which receive the first event, then run automatically.
type AutoFSM interface {
	ConfigState(State) ConfigState
	Feed(Event)
	Start(Event)
	Stop()
	Close()
}

type stateMachine struct {
	chEvent      chan Event
	currentState State
	states       map[State]*interState
}

func newStateMachine(startState State) *stateMachine {
	fsm := &stateMachine{}
	fsm.states = make(map[State]*interState)
	fsm.currentState = startState
	//fsm.ConfigState(startState)
	return fsm
}

func NewStepFSM(startState State) StepFSM {
	return newStateMachine(startState)
}

func NewAutoFSM(startState State) AutoFSM {
	fsm := newStateMachine(startState)
	fsm.chEvent = make(chan Event, 1)
	return fsm
}

func (fsm *stateMachine) ConfigState(s State) ConfigState {
	if ss, ok := fsm.states[s]; ok {
		return ss
	} else {
		ss = &interState{}
		ss.id = s
		ss.fsm = fsm
		fsm.states[s] = ss
		return ss
	}
}

// feed the Event ev to fsm, transfer to next state
func (fsm *stateMachine) Step(ev Event) {
	if currentState, ok := fsm.states[fsm.currentState]; !ok {
		panic("no such state")
	} else if nextState, ok := currentState.next[ev]; !ok {
		panic("can not accept the event")
	} else {
		// exit current state
		if currentState.exitAction != nil {
			currentState.exitAction.Do()
		}

		// transit to next state
		ps := currentState.id
		fsm.currentState = nextState.id
		if nextState.enterFrom != nil && nextState.enterFrom[ps] != nil {
			nextState.enterFrom[ps].Do()
		} else {
			if nextState.enterAction != nil {
				nextState.enterAction.Do()
			}
		}
	}
}

// auto run fsm
func (fsm *stateMachine) Start(startEv Event) {
	if fsm.chEvent == nil {
		panic("FSM has not initialized")
	}

	fsm.chEvent <- startEv

	for ev := range fsm.chEvent {
		fsm.Step(ev)
	}
}

// feed event to fsm for auto run next step
func (fsm *stateMachine) Feed(e Event) {
	fsm.chEvent <- e
}

// stop fsm from auto run
func (fsm *stateMachine) Stop() {
	if fsm.chEvent != nil {
		close(fsm.chEvent)
		fsm.chEvent = nil
	}
}

func (fsm *stateMachine) Close() {
	fsm.Stop()
	for _, v := range fsm.states {
		v.enterFrom = nil
		v.next = nil
	}
	fsm.states = nil
	fsm.currentState = 0
}
