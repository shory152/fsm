package fsm

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

const (
	S0 State = iota
	S1
	S2
	S3
	S4
	S5
)
const (
	E0 Event = iota
	E1
	E2
	E3
	E4
	E5
)

func TestAutoFSM(t *testing.T) {
	fsm := NewAutoFSM(S0)
	s0 := fsm.ConfigState(S0)
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

	s1 := fsm.ConfigState(S1)
	s1.Accept(E3, S3)
	s1.Accept(E4, S4)
	s1.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S1")
			fsm.Feed(E3)
		}))
	s1.OnEnterFrom(S2, ActionFunc(func() {
		fmt.Println("enter S1 from S2")
		fsm.Feed(E4)
	}))

	s3 := fsm.ConfigState(S3)
	s3.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S3")
			fmt.Println("stop")
			fsm.Stop()
		}))
	s3.OnEnterFrom(S4, ActionFunc(func() {
		fmt.Println("enter S3 from S4")
		fmt.Println("stop")
		fsm.Stop()
	}))

	s2 := fsm.ConfigState(S2)
	s2.Accept(E4, S4)
	s2.Accept(E3, S3)
	s2.Accept(E1, S1)
	s2.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S2")
			if rand.Int()%3 == 0 {
				fsm.Feed(E4)
			} else if rand.Int()%3 == 1 {
				fsm.Feed(E3)
			} else {
				fsm.Feed(E1)
			}
		}))

	s4 := fsm.ConfigState(S4)
	s4.Accept(E3, S3)
	s4.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S4")
			fsm.Feed(E3)
		}))
	s4.OnEnterFrom(S1, ActionFunc(func() {
		fmt.Println("enter S4 from S1")
		fsm.Feed(E3)
	}))

	rand.Seed(time.Now().Unix())
	fsm.Start(E1)
}

func TestStepSFM(t *testing.T) {
	type job struct {
		fsm StepFSM
		nev Event
	}

	jobQ := make(chan job, 1)

	var j job
	j.fsm = NewStepFSM(S0)

	fsm := j.fsm

	s0 := fsm.ConfigState(S0)
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

	s1 := fsm.ConfigState(S1)
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

	s3 := fsm.ConfigState(S3)
	s3.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S3")
			fmt.Println("stop")
			fsm.Close()
			close(jobQ)
		}))
	s3.OnEnterFrom(S4, ActionFunc(func() {
		fmt.Println("enter S3 from S4")
		fmt.Println("stop")
		fsm.Close()
		close(jobQ)
	}))

	s2 := fsm.ConfigState(S2)
	s2.Accept(E4, S4)
	s2.Accept(E3, S3)
	s2.Accept(E1, S1)
	s2.OnEnter(
		ActionFunc(func() {
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

	s4 := fsm.ConfigState(S4)
	s4.Accept(E3, S3)
	s4.OnEnter(
		ActionFunc(func() {
			fmt.Println("enter S4")
			//fsm.Feed(E3)
			j.nev = E3
			jobQ <- j
		}))
	s4.OnEnterFrom(S1, ActionFunc(func() {
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
		_  State = iota
		S0       // start
		S1       // recv '<'
		S2       // recv '</'
		S3       // recv '<otehr'
		S4       // recv '>'
		S5       // >other
		S6       // end
		S7       // err
	)
	const (
		_  Event = iota
		E1       // blank
		E2       // '<'
		E3       // '/'
		E4       // '>'
		E5       // other char
		E6       // \t \n \r
		E7       // EOF
	)

	fsm := NewAutoFSM(S0)
	defer fsm.Close()

	s0 := fsm.ConfigState(S0)
	s0.Accept(E1, S0)
	s0.Accept(E2, S1)
	s0.Accept(E7, S6)
	s0.Accept(E5, S7)
	s0.OnEnter(ActionFunc(func() {
		if c, ok := <-chChar; !ok {
			fsm.Feed(E7)
		} else if c == '<' {
			fsm.Feed(E2)
		} else if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			fsm.Feed(E1)
		} else {
			fsm.Feed(E5)
		}
	}))

	s1 := fsm.ConfigState(S1)
	s1.Accept(E3, S2)
	s1.Accept(E5, S3)
	s1.Accept(E7, S7)
	s1.OnEnter(ActionFunc(func() {
		if c, ok := <-chChar; !ok {
			fsm.Feed(E7)
		} else if c == '/' {
			fsm.Feed(E3)
		} else {
			val.WriteRune(c)
			fsm.Feed(E5)
		}
	}))

	s2 := fsm.ConfigState(S2)
	s2.Accept(E5, S2)
	s2.Accept(E4, S4)
	s2.Accept(E7, S7)
	s2.OnEnter(ActionFunc(func() {
		if c, ok := <-chChar; !ok {
			fsm.Feed(E7)
		} else if c == '>' {
			wd := word{}
			wd.typ = WORD_TAG_CLOSE
			wd.value = val.String()
			chToken <- wd
			val.Reset()
			fsm.Feed(E4)
		} else {
			val.WriteRune(c)
			fsm.Feed(E5)
		}
	}))

	s3 := fsm.ConfigState(S3)
	s3.Accept(E7, S7)
	s3.Accept(E5, S3)
	s3.Accept(E4, S4)
	s3.OnEnter(ActionFunc(func() {
		if c, ok := <-chChar; !ok {
			fsm.Feed(E7)
		} else if c == '>' {
			wd := word{}
			wd.typ = WORD_TAG_OPEN
			wd.value = val.String()
			chToken <- wd
			val.Reset()
			fsm.Feed(E4)
		} else {
			val.WriteRune(c)
			fsm.Feed(E5)
		}
	}))

	s4 := fsm.ConfigState(S4)
	s4.Accept(E7, S6)
	s4.Accept(E5, S5)
	s4.Accept(E6, S4)
	s4.Accept(E2, S1)
	s4.OnEnter(ActionFunc(func() {
		if c, ok := <-chChar; !ok {
			fsm.Feed(E7)
		} else if c == '<' {
			fsm.Feed(E2)
		} else if c == '\n' || c == '\r' || c == '\t' {
			fsm.Feed(E6)
		} else {
			val.WriteRune(c)
			fsm.Feed(E5)
		}
	}))

	s5 := fsm.ConfigState(S5)
	s5.Accept(E5, S5)
	s5.Accept(E2, S1)
	s5.Accept(E7, S7)
	s5.Accept(E6, S7)
	s5.OnEnter(ActionFunc(func() {
		if c, ok := <-chChar; !ok {
			fsm.Feed(E7)
		} else if c == '<' {
			wd := word{}
			wd.typ = WORD_VAL
			wd.value = val.String()
			chToken <- wd
			val.Reset()
			fsm.Feed(E2)
		} else if c == '\n' || c == '\r' || c == '\t' {
			fsm.Feed(E6)
		} else {
			val.WriteRune(c)
			fsm.Feed(E5)
		}
	}))

	s6 := fsm.ConfigState(S6)
	s6.OnEnter(ActionFunc(func() {
		//fmt.Println("parse OK")
		fsm.Stop()
	}))

	s7 := fsm.ConfigState(S7)
	s7.OnEnter(ActionFunc(func() {
		fmt.Println("parse error")
		fsm.Stop()
	}))

	fsm.Start(E1)

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
		_  State = iota
		S0       // start
		S1       // recv '<'
		S2       // recv '</'
		S3       // recv '<otehr'
		S4       // recv '>'
		S5       // >other
		S6       // end
		S7       // err
	)
	const (
		_  Event = iota
		E1       // blank
		E2       // '<'
		E3       // '/'
		E4       // '>'
		E5       // other char
		E6       // \t \n \r
		E7       // EOF
	)

	fsm := NewAutoFSM(S0)
	defer fsm.Close()

	s0 := fsm.ConfigState(S0)
	s0.Accept(E1, S0)
	s0.Accept(E2, S1)
	s0.Accept(E7, S6)
	s0.Accept(E5, S7)
	s0.OnEnter(ActionFunc(func() {
		if c, ok := getchar(); !ok {
			fsm.Feed(E7)
		} else if c == '<' {
			fsm.Feed(E2)
		} else if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			fsm.Feed(E1)
		} else {
			fsm.Feed(E5)
		}
	}))

	s1 := fsm.ConfigState(S1)
	s1.Accept(E3, S2)
	s1.Accept(E5, S3)
	s1.Accept(E7, S7)
	s1.OnEnter(ActionFunc(func() {
		if c, ok := getchar(); !ok {
			fsm.Feed(E7)
		} else if c == '/' {
			fsm.Feed(E3)
		} else {
			val.WriteRune(c)
			fsm.Feed(E5)
		}
	}))

	s2 := fsm.ConfigState(S2)
	s2.Accept(E5, S2)
	s2.Accept(E4, S4)
	s2.Accept(E7, S7)
	s2.OnEnter(ActionFunc(func() {
		if c, ok := getchar(); !ok {
			fsm.Feed(E7)
		} else if c == '>' {
			wd := word{}
			wd.typ = WORD_TAG_CLOSE
			wd.value = val.String()
			out[i] = &wd
			i++
			val.Reset()
			fsm.Feed(E4)
		} else {
			val.WriteRune(c)
			fsm.Feed(E5)
		}
	}))

	s3 := fsm.ConfigState(S3)
	s3.Accept(E7, S7)
	s3.Accept(E5, S3)
	s3.Accept(E4, S4)
	s3.OnEnter(ActionFunc(func() {
		if c, ok := getchar(); !ok {
			fsm.Feed(E7)
		} else if c == '>' {
			wd := word{}
			wd.typ = WORD_TAG_OPEN
			wd.value = val.String()
			out[i] = &wd
			i++
			val.Reset()
			fsm.Feed(E4)
		} else {
			val.WriteRune(c)
			fsm.Feed(E5)
		}
	}))

	s4 := fsm.ConfigState(S4)
	s4.Accept(E7, S6)
	s4.Accept(E5, S5)
	s4.Accept(E6, S4)
	s4.Accept(E2, S1)
	s4.OnEnter(ActionFunc(func() {
		if c, ok := getchar(); !ok {
			fsm.Feed(E7)
		} else if c == '<' {
			fsm.Feed(E2)
		} else if c == '\n' || c == '\r' || c == '\t' {
			fsm.Feed(E6)
		} else {
			val.WriteRune(c)
			fsm.Feed(E5)
		}
	}))

	s5 := fsm.ConfigState(S5)
	s5.Accept(E5, S5)
	s5.Accept(E2, S1)
	s5.Accept(E7, S7)
	s5.Accept(E6, S7)
	s5.OnEnter(ActionFunc(func() {
		if c, ok := getchar(); !ok {
			fsm.Feed(E7)
		} else if c == '<' {
			wd := word{}
			wd.typ = WORD_VAL
			wd.value = val.String()
			out[i] = &wd
			i++
			val.Reset()
			fsm.Feed(E2)
		} else if c == '\n' || c == '\r' || c == '\t' {
			fsm.Feed(E6)
		} else {
			val.WriteRune(c)
			fsm.Feed(E5)
		}
	}))

	s6 := fsm.ConfigState(S6)
	s6.OnEnter(ActionFunc(func() {
		//fmt.Println("parse OK")
		fsm.Stop()
	}))

	s7 := fsm.ConfigState(S7)
	s7.OnEnter(ActionFunc(func() {
		fmt.Println("parse error")
		fsm.Stop()
	}))

	fsm.Start(E1)

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
