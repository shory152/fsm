package xml

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/shory152/fsm"
)

const (
	_             int = iota
	XML_TAG_OPTN      // <name>
	XML_TEXT          // between openTag and closeTag
	XML_TAG_CLOSE     // </name>
	XML_PRO_KEY       // <xx KEY=v1>
	XML_PRO_VAL       // <xx k1=VALUE>
	XML_COMMENT       // <!-- ... -->
	XML_HEAD          // <?xml ...?>
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
		_       fsm.State = iota
		S_start           // start
		S_lt              // recv '<'
		S_h1              // header <?
		S_h2              // header <?xml
		S_h3              // header <?xml ... ?>
		S_ct1             // close tag '</'
		S_ct2             // close tag '</xxx>'
		S_ot1             // open tag <xxx
		S_ot2             //
		S_ot3             //
		S_tt              // text
		S_pt1             // k=v
		S_pt2             //
		S_cm1             // comment <!
		S_cm2             //
		S_cm3             //
		S_cm4             //
		S_cm5             //
		S_cm6             //
		S_err             //
		S_serr            // syntax error
	)
	const (
		_       fsm.Event = iota
		E_start           // start fsm
		E_blank           // blank
		E_line            // \t \n \r
		E_space           // ' '
		E_lt              // '<'
		E_gt              // '>'
		E_eq              // '='
		E_qes             // '?'
		E_sl              // '/'
		E_gth             // '!'
		E_desh            // '-'
		E_oc              // other char
		E_err             // read error
		E_serr            // syntax error
	)

	var val bytes.Buffer
	nextToken := XmlToken{}
	fgStopped := false
	fgStarted := false
	var errAction error
	var syntaxErrOff int64
	var hasHeader bool

	sm := fsm.NewAutoFSM(S_start)

	// conf S_start
	s0 := sm.ConfigState(S_start)
	s0.Accept(E_start, S_start)
	s0.Accept(E_blank, S_start)
	s0.Accept(E_lt, S_lt)
	s0.Accept(E_err, S_err)
	s0.Accept(E_serr, S_serr)
	s0.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '<' {
			sm.Feed(E_lt)
		} else if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			sm.Feed(E_blank)
		} else {
			syntaxErrOff = xmlr.Size() - int64(xmlr.Len())
			sm.Feed(E_serr)
		}
	}))

	serr := sm.ConfigState(S_err)
	serr.OnEnter(fsm.ActionFunc(func() {
		sm.Stop()
	}))

	serr2 := sm.ConfigState(S_serr)
	serr2.OnEnter(fsm.ActionFunc(func() {
		syntaxErrOff = xmlr.Size() - int64(xmlr.Len())
		tmp := xml[syntaxErrOff:]
		if len(tmp) > 16 {
			tmp = tmp[:16]
		}
		errAction = fmt.Errorf("syntax error: at %v, before %v", syntaxErrOff, tmp)
		sm.Stop()
	}))

	s1 := sm.ConfigState(S_lt)
	s1.Accept(E_err, S_err)
	s1.Accept(E_serr, S_serr)
	s1.Accept(E_qes, S_h1)
	s1.Accept(E_sl, S_ct1)
	s1.Accept(E_gth, S_cm1)
	s1.Accept(E_oc, S_ot1)
	s1.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '?' {
			if !hasHeader {
				hasHeader = true
				sm.Feed(E_qes)
			} else {
				syntaxErrOff = xmlr.Size() - int64(xmlr.Len())
				sm.Feed(E_serr)
			}
		} else if c == '/' {
			sm.Feed(E_sl)
		} else if c == '!' {
			sm.Feed(E_gth)
		} else {
			val.WriteRune(c)
			sm.Feed(E_oc)
		}
	}))

	h1 := sm.ConfigState(S_h1)
	h1.Accept(E_err, S_err)
	h1.Accept(E_serr, S_serr)
	h1.Accept(E_qes, S_h2)
	h1.Accept(E_oc, S_h1)
	h1.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '?' {
			sm.Feed(E_qes)
		} else if c == '>' || c == '<' {
			syntaxErrOff = xmlr.Size() - int64(xmlr.Len())
			sm.Feed(E_serr)
		} else {
			val.WriteRune(c)
			sm.Feed(E_oc)
		}
	}))

	h2 := sm.ConfigState(S_h2)
	h2.Accept(E_err, S_err)
	h2.Accept(E_gt, S_h3)
	h2.Accept(E_oc, S_h1)
	h2.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '>' {
			nextToken = XmlToken{XML_HEAD, val.String()}
			val.Reset()
			sm.Pause(E_gt)
		} else {
			val.WriteRune(c)
			sm.Feed(E_oc)
		}
	}))

	h3 := sm.ConfigState(S_h3)
	h3.Accept(E_err, S_err)
	h3.Accept(E_serr, S_serr)
	h3.Accept(E_blank, S_h3)
	h3.Accept(E_lt, S_lt)
	h3.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '<' {
			sm.Feed(E_lt)
		} else if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			sm.Feed(E_blank)
		} else {
			syntaxErrOff = xmlr.Size() - int64(xmlr.Len())
			sm.Feed(E_serr)
		}
	}))

	ct1 := sm.ConfigState(S_ct1)
	ct1.Accept(E_err, S_err)
	ct1.Accept(E_serr, S_serr)
	ct1.Accept(E_gt, S_ct2)
	ct1.Accept(E_oc, S_ct1)
	ct1.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '>' {
			nextToken = XmlToken{XML_TAG_CLOSE, val.String()}
			val.Reset()
			sm.Pause(E_gt)
		} else {
			val.WriteRune(c)
			sm.Feed(E_oc)
		}
	}))

	ct2 := sm.ConfigState(S_ct2)
	ct2.Accept(E_err, S_err)
	ct2.Accept(E_serr, S_serr)
	ct2.Accept(E_blank, S_ct2)
	ct2.Accept(E_lt, S_lt)
	ct2.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '<' {
			sm.Feed(E_lt)
		} else if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			sm.Feed(E_blank)
		} else {
			syntaxErrOff = xmlr.Size() - int64(xmlr.Len())
			sm.Feed(E_serr)
		}
	}))

	ot1 := sm.ConfigState(S_ot1)
	ot1.Accept(E_err, S_err)
	ot1.Accept(E_serr, S_serr)
	ot1.Accept(E_gt, S_ot2)
	ot1.Accept(E_space, S_pt1)
	ot1.Accept(E_oc, S_ot1)
	ot1.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '>' {
			nextToken = XmlToken{XML_TAG_OPTN, val.String()}
			val.Reset()
			sm.Pause(E_gt)
		} else if c == ' ' {
			nextToken = XmlToken{XML_TAG_OPTN, val.String()}
			val.Reset()
			sm.Pause(E_space)
		} else {
			val.WriteRune(c)
			sm.Feed(E_oc)
		}
	}))

	ot2 := sm.ConfigState(S_ot2)
	ot2.Accept(E_err, S_err)
	ot2.Accept(E_serr, S_serr)
	ot2.Accept(E_line, S_ot2)
	ot2.Accept(E_lt, S_lt)
	ot2.Accept(E_oc, S_tt)
	ot2.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '\n' || c == '\r' || c == '\t' {
			sm.Feed(E_line)
		} else if c == '<' {
			sm.Feed(E_lt)
		} else {
			val.WriteRune(c)
			sm.Feed(E_oc)
		}
	}))

	tt := sm.ConfigState(S_tt)
	tt.Accept(E_err, S_err)
	tt.Accept(E_serr, S_serr)
	tt.Accept(E_oc, S_tt)
	tt.Accept(E_lt, S_lt) // output text
	tt.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '<' {
			nextToken = XmlToken{XML_TEXT, val.String()}
			val.Reset()
			sm.Pause(E_lt)
		} else {
			val.WriteRune(c)
			sm.Feed(E_oc)
		}
	}))

	pt1 := sm.ConfigState(S_pt1)
	pt1.Accept(E_err, S_err)
	pt1.Accept(E_serr, S_serr)
	pt1.Accept(E_oc, S_pt1)
	pt1.Accept(E_eq, S_pt2) // output key
	pt1.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '=' {
			nextToken = XmlToken{XML_PRO_KEY, val.String()}
			val.Reset()
			sm.Pause(E_eq)
		} else {
			val.WriteRune(c)
			sm.Feed(E_oc)
		}
	}))

	pt2 := sm.ConfigState(S_pt2)
	pt2.Accept(E_err, S_err)
	pt2.Accept(E_serr, S_serr)
	pt2.Accept(E_oc, S_pt2)
	pt2.Accept(E_space, S_pt1) // output val
	pt2.Accept(E_gt, S_ot2)    // output val
	pt2.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == ' ' {
			nextToken = XmlToken{XML_PRO_VAL, val.String()}
			val.Reset()
			sm.Pause(E_space)
		} else if c == '>' {
			nextToken = XmlToken{XML_PRO_VAL, val.String()}
			val.Reset()
			sm.Pause(E_gt)
		} else {
			val.WriteRune(c)
			sm.Feed(E_oc)
		}
	}))

	cm1 := sm.ConfigState(S_cm1)
	cm1.Accept(E_err, S_err)
	cm1.Accept(E_serr, S_serr)
	cm1.Accept(E_desh, S_cm2)
	cm1.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '-' {
			sm.Feed(E_desh)
		} else {
			syntaxErrOff = xmlr.Size() - int64(xmlr.Len())
			sm.Feed(E_serr)
		}
	}))

	cm2 := sm.ConfigState(S_cm2)
	cm2.Accept(E_err, S_err)
	cm2.Accept(E_serr, S_serr)
	cm2.Accept(E_desh, S_cm3)
	cm2.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '-' {
			sm.Feed(E_desh)
		} else {
			syntaxErrOff = xmlr.Size() - int64(xmlr.Len())
			sm.Feed(E_serr)
		}
	}))

	cm3 := sm.ConfigState(S_cm3)
	cm3.Accept(E_err, S_err)
	cm3.Accept(E_serr, S_serr)
	cm3.Accept(E_desh, S_cm4)
	cm3.Accept(E_oc, S_cm3)
	cm3.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '-' {
			sm.Feed(E_desh)
		} else if c == '<' || c == '>' {
			syntaxErrOff = xmlr.Size() - int64(xmlr.Len())
			sm.Feed(E_serr)
		} else {
			val.WriteRune(c)
			sm.Feed(E_oc)
		}
	}))

	cm4 := sm.ConfigState(S_cm4)
	cm4.Accept(E_err, S_err)
	cm4.Accept(E_serr, S_serr)
	cm4.Accept(E_desh, S_cm5)
	cm4.Accept(E_oc, S_cm3)
	cm4.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '-' {
			sm.Feed(E_desh)
		} else {
			val.WriteRune('-')
			val.WriteRune(c)
			sm.Feed(E_oc)
		}
	}))

	cm5 := sm.ConfigState(S_cm5)
	cm5.Accept(E_err, S_err)
	cm5.Accept(E_serr, S_serr)
	cm5.Accept(E_gt, S_cm6) // out comment
	cm5.Accept(E_oc, S_cm3)
	cm5.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '>' {
			nextToken = XmlToken{XML_COMMENT, val.String()}
			val.Reset()
			sm.Pause(E_gt)
		} else {
			val.WriteString("--")
			val.WriteRune(c)
			sm.Feed(E_oc)
		}
	}))

	cm6 := sm.ConfigState(S_cm6)
	cm6.Accept(E_err, S_err)
	cm6.Accept(E_serr, S_serr)
	cm6.Accept(E_blank, S_cm6)
	cm6.Accept(E_lt, S_lt)
	cm6.OnEnter(fsm.ActionFunc(func() {
		if c, _, err := xmlr.ReadRune(); err != nil {
			errAction = err
			sm.Feed(E_err)
		} else if c == '<' {
			sm.Feed(E_lt)
		} else if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			sm.Feed(E_blank)
		} else {
			syntaxErrOff = xmlr.Size() - int64(xmlr.Len())
			sm.Feed(E_serr)
		}
	}))

	return XmlScanner(func() (XmlToken, error) {
		if fgStopped {
			sm.Close()
			return XmlToken{}, errAction
		}
		if !fgStarted {
			fgStarted = true
			sm.Start(E_start)
		} else {
			sm.Resume()
		}

		return nextToken, errAction
	})
}

var xmlstr string = `
<?xml version=1.0 encoding=UTF-8 ?>
<books>
	<!-- 2 books - - -x- -- -->
	<book p1="v1" p2="v2">
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

func TestXml(t *testing.T) {
	scanner := scanXml(xmlstr)
	for {
		if tk, err := scanner(); err != nil {
			if err != io.EOF {
				fmt.Println(err)
			}

			break
		} else {
			fmt.Println(tk)
		}
	}
}
