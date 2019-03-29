package test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/shory152/fsm"
)

const (
	_             int = iota
	XML_TAG_OPTN      // <name>
	XML_TEXT          // between openTag and closeTag
	XML_TAG_CLOSE     // </name>
	XML_HEAD          // <?xml ...?>
	XML_COMMENT       // <!-- ... -->
)

type XmlToken struct {
	ID  int
	Val string
}

type XmlScanner func() (XmlToken, error)

func scanXml(xml string) XmlScanner {
	xmlr := bytes.NewReader([]byte(xml))

	// define fsm for scanning a token
	const (
		_  fsm.State = iota
		S0           // start
		S1           // recv '<'
		S2           // recv '</'
		S3           // recv '<otehr'
		S4           // recv '>'
		S5           // >other
		S6           // end for read error
		S7           // syntax error
	)
	const (
		_  fsm.Event = iota
		E1           // blank
		E2           // '<'
		E3           // '/'
		E4           // '>'
		E5           // other char
		E6           // \t \n \r
		E7           // EOF or other error
	)

	var val bytes.Buffer
	nextEvent := fsm.Event(E1)
	nextToken := XmlToken{}
	fgStopped := false
	fgStarted := false
	var errAction error

	sm := fsm.NewAutoFSM(S0)

	s0 := sm.ConfigState(S0)
	s0.Accept(E1, S0)
	s0.Accept(E2, S1)
	s0.Accept(E7, S6)
	s0.Accept(E5, S7)
	s0.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
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
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
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
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E7)
		} else if c == '>' {
			nextToken = XmlToken{XML_TAG_CLOSE, val.String()}
			val.Reset()
			nextEvent = E4
			sm.Pause(E4)
			//sm.Feed(E4)
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
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E7)
		} else if c == '>' {
			nextToken = XmlToken{XML_TAG_OPTN, val.String()}
			val.Reset()
			//sm.Feed(E4)
			nextEvent = E4
			sm.Pause(E4)
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
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
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
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E7)
		} else if c == '<' {
			nextToken = XmlToken{XML_TEXT, val.String()}
			val.Reset()
			//sm.Feed(E2)
			nextEvent = E2
			sm.Pause(E2)
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
		if errAction != nil && errAction != io.EOF {
			fmt.Println("parse error:", errAction)
		}
		fgStopped = true
		sm.Stop()
	}))

	s7 := sm.ConfigState(S7)
	s7.OnEnter(fsm.ActionFunc(func() {
		fmt.Println("parse error: syntax error")
		errAction = errors.New("syntax error")
		sm.Stop()
		fgStopped = true
	}))

	return XmlScanner(func() (XmlToken, error) {
		if fgStopped {
			sm.Close()
			return XmlToken{}, errAction
		}
		if !fgStarted {
			fgStarted = true
			sm.Start(nextEvent)
		} else {
			sm.Resume()
		}

		return nextToken, errAction
	})
}

var xmlstr string = `
<xml version = 1.0>
<books>
	<book>
		<name>
		
		 大 道 中 国 </name>
		<price> 89.00 </price>
		<author>张大中</author>
	</book>
	
	<book>
		<name>小猪唏哩呼噜</name>
		<price>22.50</price>
		<author>Alex</author>
	</book>
</books>
`

func Test2(t *testing.T) {
	scanner := scanXml(xmlstr)

	for {
		if tk, err := scanner(); err != nil {
			fmt.Println("error:", err)
			break
		} else {
			fmt.Println(tk)
		}
	}
	fmt.Println(scanner())

	N := 100000
	btm := time.Now()
	for i := 0; i < N; i++ {
		scanner = scanXml(xmlstr)
		for {
			if tk, err := scanner(); err != nil {
				if err != io.EOF {
					fmt.Println("error:", err)
				}
				break
			} else {
				_ = tk
			}
		}
	}
	elapse := time.Since(btm)
	fmt.Printf("%f ns/op\n", float64(elapse.Nanoseconds())/float64(N))

}

type XmlNode struct {
	name string
	text string
	kids []*XmlNode
}

func parseTree(scanner XmlScanner, parent *XmlNode) *XmlNode {
	for {
		if tk, err := scanner(); err != nil {
			fmt.Println(err)
			break
		} else {
			switch tk.ID {
			case XML_TAG_OPTN:
				nodeTag := &XmlNode{}
				nodeTag.name = tk.Val
				if parent == nil {
					parent = nodeTag
				} else {
					parent.kids = append(parent.kids, nodeTag)
				}
				parseTree(scanner, nodeTag)
			case XML_TAG_CLOSE:
				if parent.name != tk.Val {
					fmt.Println(parent, tk)
					panic("not well-formed xml")
				}
				return parent
			case XML_TEXT:
				if parent == nil {
					panic("no parent node for TEXT")
				}
				parent.text = tk.Val
			default:
				panic("invalid XmlToken")
			}
		}
	}
	return parent
}

func showXmlTree(root *XmlNode, lvl int) {
	if root == nil {
		return
	}
	for i := 0; i < lvl; i++ {
		fmt.Printf("  ")
	}
	fmt.Printf("+- ")
	if len(root.kids) > 0 {
		fmt.Printf("{%v, %v kids}\n", root.name, len(root.kids))
	} else {
		fmt.Printf("{%v, %v}\n", root.name, root.text)
	}

	for _, c := range root.kids {
		showXmlTree(c, lvl+1)
	}
}

func TestParseXmlTree(t *testing.T) {
	scanner := scanXml(xmlstr)
	root := parseTree(scanner, nil)
	showXmlTree(root, 1)
}
