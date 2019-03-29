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
	OnExitEvent(e Event, a Action) ConfigState
}

type interState struct {
	id          State
	enterAction Action
	enterFrom   map[State]Action
	exitAction  Action
	exitFrom    map[Event]Action
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

// execute act when exit this state triggered by Event e
func (is *interState) OnExitEvent(e Event, act Action) ConfigState {
	is.exitFrom[e] = act
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
	ConfigState(s State) ConfigState
	Feed(next Event)
	Start(start Event)
	Stop()
	Pause(next Event)
	Resume()
	Close()
}

type stateMachine struct {
	flag         uint32
	nextEvent    Event
	currentState State
	states       map[State]*interState
}

const (
	fsm_flag_auto uint32 = 1 << iota
	fsm_flag_step
	fsm_flag_running
	fsm_flag_pause
	fsm_flag_stopped
	fsm_flag_nextev
)

func (fsm *stateMachine) isAutoFsm() bool {
	return fsm.flag&fsm_flag_auto > 0
}
func (fsm *stateMachine) isStepFsm() bool {
	return fsm.flag&fsm_flag_step > 0
}
func (fsm *stateMachine) isRunning() bool {
	return fsm.flag&fsm_flag_running > 0
}
func (fsm *stateMachine) isStopped() bool {
	return fsm.flag&fsm_flag_stopped > 0
}
func (fsm *stateMachine) isPaused() bool {
	return fsm.flag&fsm_flag_pause > 0
}
func (fsm *stateMachine) isSetNextEv() bool {
	return fsm.flag&fsm_flag_nextev > 0
}

func newStateMachine(startState State) *stateMachine {
	fsm := &stateMachine{}
	fsm.states = make(map[State]*interState)
	fsm.currentState = startState
	//fsm.ConfigState(startState)
	return fsm
}

func NewStepFSM(startState State) StepFSM {
	fsm := newStateMachine(startState)
	fsm.flag |= fsm_flag_step
	return fsm
}

func NewAutoFSM(startState State) AutoFSM {
	fsm := newStateMachine(startState)
	fsm.flag |= fsm_flag_auto
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
		if currentState.exitFrom != nil && currentState.exitFrom[ev] != nil {
			currentState.exitFrom[ev].Do()
		} else if currentState.exitAction != nil {
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

func (fsm *stateMachine) autoRun() {
	for fsm.isSetNextEv() {
		fsm.flag &= ^fsm_flag_nextev
		fsm.Step(fsm.nextEvent)
		if fsm.isPaused() || fsm.isStopped() {
			break
		}
	}
}

// auto run fsm
func (fsm *stateMachine) Start(startEv Event) {

	if fsm.isStopped() {
		panic("FSM has been stopped")
	}
	if fsm.isRunning() {
		panic("FSM has started")
	}

	fsm.flag |= fsm_flag_running
	fsm.flag &= ^fsm_flag_pause
	fsm.Feed(startEv)
	fsm.autoRun()
}

// feed event to fsm for auto run next step
func (fsm *stateMachine) Feed(e Event) {
	fsm.nextEvent = e
	fsm.flag |= fsm_flag_nextev
}

// stop fsm from auto run
func (fsm *stateMachine) Stop() {
	fsm.flag &= ^fsm_flag_running
	fsm.flag |= fsm_flag_stopped
}

// pause fsm from auto run
func (fsm *stateMachine) Pause(next Event) {
	fsm.flag &= ^fsm_flag_running
	fsm.flag |= fsm_flag_pause
	fsm.Feed(next)
}

// resume fsm
func (fsm *stateMachine) Resume() {
	if fsm.isStopped() {
		panic("FSM has been stopped")
	}
	if fsm.isRunning() {
		panic("FSM has started")
	}
	if !fsm.isPaused() {
		panic("FSM has not paused")
	}

	fsm.flag &= ^fsm_flag_pause
	fsm.flag |= fsm_flag_running
	fsm.autoRun()
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
