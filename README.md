# fsm
A state machine

sample:

```
import "github.com/shory152/fsm"

const (
	S0 fsm.State = iota
	S1
	S2
	S3
	S4
	S5
)
const (
	E0 fsm.Event = iota
	E1
	E2
	E3
	E4
	E5
)

func testAutoFSM() {
	sm := fsm.NewAutoFSM(S0)
	defer sm.Close()
  
	s0 := sm.ConfigState(S0)
	s0.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S0")
		}))
	s0.OnExit(
		ActionFunc(func() {
			fmt.Println("exit S0")
		}))
	s0.Accept(E1, S1)
	s0.Accept(E2, S2)

	s1 := sm.ConfigState(S1)
	s1.Accept(E3, S3)
	s1.Accept(E4, S4)
	s1.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S1")
			sm.Feed(E3)
		}))
	s1.OnEnterFrom(S2, ActionFunc(func() {
		fmt.Println("enter S1 from S2")
		sm.Feed(E4)
	}))

	s3 := sm.ConfigState(S3)
	s3.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S3")
			fmt.Println("stop")
			sm.Stop()
		}))
	s3.OnEnterFrom(S4, ActionFunc(func() {
		fmt.Println("enter S3 from S4")
		fmt.Println("stop")
		sm.Stop()
	}))

	s2 := sm.ConfigState(S2)
	s2.Accept(E4, S4)
	s2.Accept(E3, S3)
	s2.Accept(E1, S1)
	s2.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S2")
			if rand.Int()%3 == 0 {
				sm.Feed(E4)
			} else if rand.Int()%3 == 1 {
				sm.Feed(E3)
			} else {
				sm.Feed(E1)
			}
		}))

	s4 := sm.ConfigState(S4)
	s4.Accept(E3, S3)
	s4.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S4")
			sm.Feed(E3)
		}))
	s4.OnEnterFrom(S1, ActionFunc(func() {
		fmt.Println("enter S4 from S1")
		sm.Feed(E3)
	}))

	rand.Seed(time.Now().Unix())
	sm.Start(E1)
}

func testStepFSM() {
	type job struct {
		fsm StepFSM
		nev Event
	}

	jobQ := make(chan job, 1)

	var j job
	sm = fsm.NewStepFSM(S0)
	defer sm.Close()

	j.fsm = sm

	s0 := sm.ConfigState(S0)
	s0.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S0") // never call S0.OnEnter Action
		}))
	s0.OnExit(
		ActionFunc(func() {
			fmt.Println("exit S0")
		}))
	s0.Accept(E1, S1)
	s0.Accept(E2, S2)

	s1 := sm.ConfigState(S1)
	s1.Accept(E3, S3)
	s1.Accept(E4, S4)
	s1.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S1")
			j.nev = E3
			jobQ <- j
		}))
	s1.OnEnterFrom(S2, ActionFunc(func() {
		fmt.Println("enter S1 from S2")
		j.nev = E4
		jobQ <- j
	}))

	s3 := sm.ConfigState(S3)
	s3.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S3")
			fmt.Println("stop")
			close(jobQ)
		}))
	s3.OnEnterFrom(S4, ActionFunc(func() {
		fmt.Println("enter S3 from S4")
		fmt.Println("stop")
		close(jobQ)
	}))

	s2 := sm.ConfigState(S2)
	s2.Accept(E4, S4)
	s2.Accept(E3, S3)
	s2.Accept(E1, S1)
	s2.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S2")
			if rand.Int()%3 == 0 {
				j.nev = E4
				jobQ <- j
			} else if rand.Int()%3 == 1 {
				j.nev = E3
				jobQ <- j
			} else {
				j.nev = E1
				jobQ <- j
			}
		}))

	s4 := sm.ConfigState(S4)
	s4.Accept(E3, S3)
	s4.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S4")
			j.nev = E3
			jobQ <- j
		}))
	s4.OnEnterFrom(S1, ActionFunc(func() {
		fmt.Println("enter S4 from S1")
		j.nev = E3
		jobQ <- j
	}))

	rand.Seed(time.Now().Unix())
	j.nev = E1
	jobQ <- j
	for j2 := range jobQ {
		j2.fsm.Step(j.nev)
	}
}
```
