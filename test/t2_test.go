package test

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/shory152/fsm"
)

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

// S0 -> ... -> S3
//
//    | S0 | S1 | S2 | S3 | S4
//----+----+----+----+----+----
// S0 |    | E1 | E2 |    |
// S1 |    |    |    | E3 | E4
// S2 |    | E1 |    | E3 | E4
// S3 |    |    |    |    |
// S4 |    |    |    | E3 |
//
func TestAutoFSM(t *testing.T) {
	sm := fsm.NewAutoFSM(S0)
	defer sm.Close()

	s0 := sm.ConfigState(S0)
	s0.OnEnter(
		fsm.ActionFunc(func() {
			fmt.Println("enter S0")
		}))
	s0.OnExit(
		fsm.ActionFunc(func() {
			fmt.Println("exit S0")
		}))
	s0.Accept(E1, S1)
	s0.Accept(E2, S2)

	s1 := sm.ConfigState(S1)
	s1.Accept(E3, S3)
	s1.Accept(E4, S4)
	s1.OnEnter(
		fsm.ActionFunc(func() {
			fmt.Println("enter S1")
			sm.Feed(E3)
		}))
	s1.OnEnterFrom(S2, fsm.ActionFunc(func() {
		fmt.Println("enter S1 from S2")
		sm.Feed(E4)
	}))

	s3 := sm.ConfigState(S3)
	s3.OnEnter(
		fsm.ActionFunc(func() {
			fmt.Println("enter S3")
			fmt.Println("stop")
			sm.Stop()
		}))
	s3.OnEnterFrom(S4, fsm.ActionFunc(func() {
		fmt.Println("enter S3 from S4")
		fmt.Println("stop")
		sm.Stop()
	}))

	s2 := sm.ConfigState(S2)
	s2.Accept(E4, S4)
	s2.Accept(E3, S3)
	s2.Accept(E1, S1)
	s2.OnEnter(
		fsm.ActionFunc(func() {
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
		fsm.ActionFunc(func() {
			fmt.Println("enter S4")
			sm.Feed(E3)
		}))
	s4.OnEnterFrom(S1, fsm.ActionFunc(func() {
		fmt.Println("enter S4 from S1")
		sm.Feed(E3)
	}))

	rand.Seed(time.Now().Unix())
	sm.Start(E1)
}

func TestStepSFM(t *testing.T) {
	type job struct {
		fsm fsm.StepFSM
		nev fsm.Event
	}

	jobQ := make(chan job, 1)

	var j job
	sm := fsm.NewStepFSM(S0)
	defer sm.Close()

	j.fsm = sm

	s0 := sm.ConfigState(S0)
	s0.OnEnter(
		fsm.ActionFunc(func() {
			fmt.Println("enter S0") // never call S0.OnEnter Action
		}))
	s0.OnExit(
		fsm.ActionFunc(func() {
			fmt.Println("exit S0")
		}))
	s0.Accept(E1, S1)
	s0.Accept(E2, S2)

	s1 := sm.ConfigState(S1)
	s1.Accept(E3, S3)
	s1.Accept(E4, S4)
	s1.OnEnter(
		fsm.ActionFunc(func() {
			fmt.Println("enter S1")
			j.nev = E3
			jobQ <- j
		}))
	s1.OnEnterFrom(S2, fsm.ActionFunc(func() {
		fmt.Println("enter S1 from S2")
		j.nev = E4
		jobQ <- j
	}))

	s3 := sm.ConfigState(S3)
	s3.OnEnter(
		fsm.ActionFunc(func() {
			fmt.Println("enter S3")
			fmt.Println("stop")
			close(jobQ)
		}))
	s3.OnEnterFrom(S4, fsm.ActionFunc(func() {
		fmt.Println("enter S3 from S4")
		fmt.Println("stop")
		close(jobQ)
	}))

	s2 := sm.ConfigState(S2)
	s2.Accept(E4, S4)
	s2.Accept(E3, S3)
	s2.Accept(E1, S1)
	s2.OnEnter(
		fsm.ActionFunc(func() {
			fmt.Println("enter S2")
			if rand.Int()%3 == 0 {
				//fsm.Feed(E4)
				j.nev = E4
				jobQ <- j
			} else if rand.Int()%3 == 1 {
				//fsm.Feed(E3)
				j.nev = E3
				jobQ <- j
			} else {
				//fsm.Feed(E1)
				j.nev = E1
				jobQ <- j
			}
		}))

	s4 := sm.ConfigState(S4)
	s4.Accept(E3, S3)
	s4.OnEnter(
		fsm.ActionFunc(func() {
			fmt.Println("enter S4")
			//fsm.Feed(E3)
			j.nev = E3
			jobQ <- j
		}))
	s4.OnEnterFrom(S1, fsm.ActionFunc(func() {
		fmt.Println("enter S4 from S1")
		//fsm.Feed(E3)
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

const (
	_              int = iota
	WORD_TAG_OPEN      // openTag: <name>
	WORD_TAG_CLOSE     // closeTag: </name>
	WORD_VAL           // text between optnTag and closeTag
)

type word struct {
	typ   int
	value string
}

func fsmparsexml(xml string, chToken chan<- word) {
	chChar := make(chan rune, 1024)
	go func() {
		for _, c := range xml {
			chChar <- c
		}
		close(chChar)
	}()

	var val bytes.Buffer

	const (
		_  fsm.State = iota
		S0           // start
		S1           // recv '<'
		S2           // recv '</'
		S3           // recv '<otehr'
		S4           // recv '>'
		S5           // >other
		S6           // end
		S7           // err
	)
	const (
		_  fsm.Event = iota
		E1           // blank
		E2           // '<'
		E3           // '/'
		E4           // '>'
		E5           // other char
		E6           // \t \n \r
		E7           // EOF
	)

	sm := fsm.NewAutoFSM(S0)
	defer sm.Close()

	s0 := sm.ConfigState(S0)
	s0.Accept(E1, S0)
	s0.Accept(E2, S1)
	s0.Accept(E7, S6)
	s0.Accept(E5, S7)
	s0.OnEnter(fsm.ActionFunc(func() {
		if c, ok := <-chChar; !ok {
			sm.Feed(E7)
		} else if c == '<' {
			sm.Feed(E2)
		} else if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			sm.Feed(E1)
		} else {
			sm.Feed(E5)
		}
	}))

	s1 := sm.ConfigState(S1)
	s1.Accept(E3, S2)
	s1.Accept(E5, S3)
	s1.Accept(E7, S7)
	s1.OnEnter(fsm.ActionFunc(func() {
		if c, ok := <-chChar; !ok {
			sm.Feed(E7)
		} else if c == '/' {
			sm.Feed(E3)
		} else {
			val.WriteRune(c)
			sm.Feed(E5)
		}
	}))

	s2 := sm.ConfigState(S2)
	s2.Accept(E5, S2)
	s2.Accept(E4, S4)
	s2.Accept(E7, S7)
	s2.OnEnter(fsm.ActionFunc(func() {
		if c, ok := <-chChar; !ok {
			sm.Feed(E7)
		} else if c == '>' {
			wd := word{}
			wd.typ = WORD_TAG_CLOSE
			wd.value = val.String()
			chToken <- wd
			val.Reset()
			sm.Feed(E4)
		} else {
			val.WriteRune(c)
			sm.Feed(E5)
		}
	}))

	s3 := sm.ConfigState(S3)
	s3.Accept(E7, S7)
	s3.Accept(E5, S3)
	s3.Accept(E4, S4)
	s3.OnEnter(fsm.ActionFunc(func() {
		if c, ok := <-chChar; !ok {
			sm.Feed(E7)
		} else if c == '>' {
			wd := word{}
			wd.typ = WORD_TAG_OPEN
			wd.value = val.String()
			chToken <- wd
			val.Reset()
			sm.Feed(E4)
		} else {
			val.WriteRune(c)
			sm.Feed(E5)
		}
	}))

	s4 := sm.ConfigState(S4)
	s4.Accept(E7, S6)
	s4.Accept(E5, S5)
	s4.Accept(E6, S4)
	s4.Accept(E2, S1)
	s4.OnEnter(fsm.ActionFunc(func() {
		if c, ok := <-chChar; !ok {
			sm.Feed(E7)
		} else if c == '<' {
			sm.Feed(E2)
		} else if c == '\n' || c == '\r' || c == '\t' {
			sm.Feed(E6)
		} else {
			val.WriteRune(c)
			sm.Feed(E5)
		}
	}))

	s5 := sm.ConfigState(S5)
	s5.Accept(E5, S5)
	s5.Accept(E2, S1)
	s5.Accept(E7, S7)
	s5.Accept(E6, S7)
	s5.OnEnter(fsm.ActionFunc(func() {
		if c, ok := <-chChar; !ok {
			sm.Feed(E7)
		} else if c == '<' {
			wd := word{}
			wd.typ = WORD_VAL
			wd.value = val.String()
			chToken <- wd
			val.Reset()
			sm.Feed(E2)
		} else if c == '\n' || c == '\r' || c == '\t' {
			sm.Feed(E6)
		} else {
			val.WriteRune(c)
			sm.Feed(E5)
		}
	}))

	s6 := sm.ConfigState(S6)
	s6.OnEnter(fsm.ActionFunc(func() {
		//fmt.Println("parse OK")
		sm.Stop()
	}))

	s7 := sm.ConfigState(S7)
	s7.OnEnter(fsm.ActionFunc(func() {
		fmt.Println("parse error")
		sm.Stop()
	}))

	sm.Start(E1)

	close(chToken)
}

func fsmparsexml2(xml string, out []*word) {

	getchar := func() func() (rune, bool) {
		buf := bytes.NewBufferString(xml)
		return func() (rune, bool) {
			if c, _, err := buf.ReadRune(); err != nil {
				return rune(0), false
			} else {
				return c, true
			}
		}
	}()

	i := 0
	var val bytes.Buffer

	const (
		_  fsm.State = iota
		S0           // start
		S1           // recv '<'
		S2           // recv '</'
		S3           // recv '<otehr'
		S4           // recv '>'
		S5           // >other
		S6           // end
		S7           // err
	)
	const (
		_  fsm.Event = iota
		E1           // blank
		E2           // '<'
		E3           // '/'
		E4           // '>'
		E5           // other char
		E6           // \t \n \r
		E7           // EOF
	)

	sm := fsm.NewAutoFSM(S0)
	defer sm.Close()

	s0 := sm.ConfigState(S0)
	s0.Accept(E1, S0)
	s0.Accept(E2, S1)
	s0.Accept(E7, S6)
	s0.Accept(E5, S7)
	s0.OnEnter(fsm.ActionFunc(func() {
		if c, ok := getchar(); !ok {
			sm.Feed(E7)
		} else if c == '<' {
			sm.Feed(E2)
		} else if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			sm.Feed(E1)
		} else {
			sm.Feed(E5)
		}
	}))

	s1 := sm.ConfigState(S1)
	s1.Accept(E3, S2)
	s1.Accept(E5, S3)
	s1.Accept(E7, S7)
	s1.OnEnter(fsm.ActionFunc(func() {
		if c, ok := getchar(); !ok {
			sm.Feed(E7)
		} else if c == '/' {
			sm.Feed(E3)
		} else {
			val.WriteRune(c)
			sm.Feed(E5)
		}
	}))

	s2 := sm.ConfigState(S2)
	s2.Accept(E5, S2)
	s2.Accept(E4, S4)
	s2.Accept(E7, S7)
	s2.OnEnter(fsm.ActionFunc(func() {
		if c, ok := getchar(); !ok {
			sm.Feed(E7)
		} else if c == '>' {
			wd := word{}
			wd.typ = WORD_TAG_CLOSE
			wd.value = val.String()
			out[i] = &wd
			i++
			val.Reset()
			sm.Feed(E4)
		} else {
			val.WriteRune(c)
			sm.Feed(E5)
		}
	}))

	s3 := sm.ConfigState(S3)
	s3.Accept(E7, S7)
	s3.Accept(E5, S3)
	s3.Accept(E4, S4)
	s3.OnEnter(fsm.ActionFunc(func() {
		if c, ok := getchar(); !ok {
			sm.Feed(E7)
		} else if c == '>' {
			wd := word{}
			wd.typ = WORD_TAG_OPEN
			wd.value = val.String()
			out[i] = &wd
			i++
			val.Reset()
			sm.Feed(E4)
		} else {
			val.WriteRune(c)
			sm.Feed(E5)
		}
	}))

	s4 := sm.ConfigState(S4)
	s4.Accept(E7, S6)
	s4.Accept(E5, S5)
	s4.Accept(E6, S4)
	s4.Accept(E2, S1)
	s4.OnEnter(fsm.ActionFunc(func() {
		if c, ok := getchar(); !ok {
			sm.Feed(E7)
		} else if c == '<' {
			sm.Feed(E2)
		} else if c == '\n' || c == '\r' || c == '\t' {
			sm.Feed(E6)
		} else {
			val.WriteRune(c)
			sm.Feed(E5)
		}
	}))

	s5 := sm.ConfigState(S5)
	s5.Accept(E5, S5)
	s5.Accept(E2, S1)
	s5.Accept(E7, S7)
	s5.Accept(E6, S7)
	s5.OnEnter(fsm.ActionFunc(func() {
		if c, ok := getchar(); !ok {
			sm.Feed(E7)
		} else if c == '<' {
			wd := word{}
			wd.typ = WORD_VAL
			wd.value = val.String()
			out[i] = &wd
			i++
			val.Reset()
			sm.Feed(E2)
		} else if c == '\n' || c == '\r' || c == '\t' {
			sm.Feed(E6)
		} else {
			val.WriteRune(c)
			sm.Feed(E5)
		}
	}))

	s6 := sm.ConfigState(S6)
	s6.OnEnter(fsm.ActionFunc(func() {
		//fmt.Println("parse OK")
		sm.Stop()
	}))

	s7 := sm.ConfigState(S7)
	s7.OnEnter(fsm.ActionFunc(func() {
		fmt.Println("parse error")
		sm.Stop()
	}))

	sm.Start(E1)

}

func TestParseXML(t *testing.T) {
	var xmlstr string = `
<xml version = 1.0>
<books>
	<book>
		<name>
		
		 大 道 中 国 </name>
		<price> 89.00 </price>
		<author>张大中</arthor>
	</book>
	
	<book>
		<name>小猪唏哩呼噜</name>
		<price>22.50</price>
		<author>Alex</arthor>
	</book>
</books>
`
	fmt.Println("----- fsm1 ------")
	chToken := make(chan word, 1)
	go fsmparsexml(xmlstr, chToken)
	for wd := range chToken {
		fmt.Println(wd)
	}
	fmt.Println("----- fsm2 ------")
	words := make([]*word, 1024)
	fsmparsexml2(xmlstr, words)
	for _, v := range words {
		if v != nil {
			fmt.Println(v)
		}
	}

	N := 100000
	btm := time.Now()
	for i := 0; i < N; i++ {
		chToken = make(chan word, 1)
		go fsmparsexml(xmlstr, chToken)
		for wd := range chToken {
			_ = wd
		}
	}
	elapse := time.Since(btm)
	fmt.Printf("%f ns/op\n", float64(elapse.Nanoseconds())/float64(N))

	btm = time.Now()
	for i := 0; i < N; i++ {
		words = make([]*word, 1024)
		fsmparsexml2(xmlstr, words)
		for wd := range words {
			_ = wd
		}
	}
	elapse = time.Since(btm)
	fmt.Printf("%f ns/op\n", float64(elapse.Nanoseconds())/float64(N))

}
