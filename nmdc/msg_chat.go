package nmdc

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func init() {
	RegisterMessage(&ChatMessage{})
	RegisterMessage(&PrivateMessage{})
	RegisterMessage(&MCTo{})
	RegisterMessage(&UserCommand{})
}

type ChatMessage struct {
	Name string
	Me   bool
	Text string
}

func (m *ChatMessage) String() string {
	if m.Name == "" {
		return m.Text
	}
	me := ""
	if m.Me {
		me = "/me "
	}
	return "<" + string(m.Name) + "> " + me + m.Text
}

func (m *ChatMessage) Type() string {
	return "" // special case
}

func (m *ChatMessage) MarshalNMDC(ec *Encoding, buf *bytes.Buffer) error {
	if m.Name != "" {
		buf.WriteByte('<')
		err := Name(m.Name).MarshalNMDC(ec, buf)
		if err != nil {
			return err
		}
		buf.WriteString("> ")
	}
	if m.Me {
		buf.WriteString("/me ")
	}
	text := m.Text
	if m.Me && strings.HasPrefix(text, "/me ") {
		text = text[4:]
	}
	err := String(text).MarshalNMDC(ec, buf)
	if err != nil {
		return err
	}
	return nil
}

func (m *ChatMessage) UnmarshalNMDC(ec *Encoding, data []byte) error {
	// <user> message text|
	// or
	// message text|
	if len(data) != 0 && data[0] == '<' {
		data = data[1:]
		var (
			i    int // index where name ends
			off  = 0 // space between the name and the message
			base = 0 // index of the previous '>' character +1
		)
		// found next '>' character and check that the next one is a whitespace
		for base < len(data) {
			j := bytes.IndexAny(data[base:], ">\r\n")
			if j < 0 || data[base+j] != '>' {
				// no '>' characters followed by a space
				// or
				// the closest character is line break
				// use the last '>' character, which is not followed by space (at index base-1)
				if base == 0 {
					return &ErrProtocolViolation{
						Err: errors.New("name in chat message should have a closing token"),
					}
				}
				i = base - 1
				off = 1
				break
			}
			if base+j == len(data)-1 {
				i = j
				off = 1
				break
			}
			r, sz := utf8.DecodeRune(data[base+j+1:])
			if unicode.IsSpace(r) {
				i = base + j
				off = sz + 1
				break
			}
			base += j + 1
		}
		name := data[:i]
		data = data[i+off:]
		if len(name) > maxName {
			return &ErrProtocolViolation{
				Err: errors.New("name in chat message is too long"),
			}
		}
		var sname Name
		if err := sname.UnmarshalNMDC(ec, name); err != nil {
			return err
		}
		m.Name = string(sname)
	}
	var text String
	if err := text.UnmarshalNMDC(ec, data); err != nil {
		return err
	}
	m.Text = string(text)
	m.Me = strings.HasPrefix(m.Text, "/me ")
	if m.Me {
		m.Text = m.Text[4:]
	}
	return nil
}

type PrivateMessage struct {
	To, From string
	Me       bool
	Name     string
	Text     string
}

func (m *PrivateMessage) Type() string {
	return "To:"
}

func (m *PrivateMessage) MarshalNMDC(ec *Encoding, buf *bytes.Buffer) error {
	if err := Name(m.To).MarshalNMDC(ec, buf); err != nil {
		return err
	}

	buf.WriteString(" From: ")
	if err := Name(m.From).MarshalNMDC(ec, buf); err != nil {
		return err
	}

	name := m.Name
	if name == "" {
		name = m.From
	}
	buf.WriteString(" $<")
	if err := Name(name).MarshalNMDC(ec, buf); err != nil {
		return err
	}
	buf.WriteString("> ")

	if m.Me {
		buf.WriteString("/me ")
	}
	text := m.Text
	if m.Me && strings.HasPrefix(text, "/me ") {
		text = text[4:]
	}
	if err := String(text).MarshalNMDC(ec, buf); err != nil {
		return err
	}
	return nil
}

func (m *PrivateMessage) UnmarshalNMDC(ec *Encoding, data []byte) error {
	var name Name

	const fromToken = " From: "
	i := bytes.Index(data, []byte(fromToken))
	if i < 0 {
		return errors.New("invalid To message: no 'from' delimiter")
	} else if err := name.UnmarshalNMDC(ec, data[:i]); err != nil {
		return err
	}
	data = data[i+len(fromToken):]
	m.To = string(name)

	const nameTokenS = " $<"
	i = bytes.Index(data, []byte(nameTokenS))
	if i < 0 {
		return errors.New("invalid To message: no name delimiter")
	} else if err := name.UnmarshalNMDC(ec, data[:i]); err != nil {
		return err
	}
	data = data[i+len(nameTokenS):]
	m.From = string(name)

	i = bytes.Index(data, []byte("> "))
	if i < 0 {
		return errors.New("invalid To message: no name end delimiter")
	} else if err := name.UnmarshalNMDC(ec, data[:i]); err != nil {
		return err
	}
	text := data[i+2:]
	m.Name = string(name)

	var s String
	if err := s.UnmarshalNMDC(ec, text); err != nil {
		return err
	}
	m.Text = string(s)
	m.Me = strings.HasPrefix(m.Text, "/me ")
	if m.Me {
		m.Text = m.Text[4:]
	}
	return nil
}

type MCTo struct {
	To, From string
	Me       bool
	Text     string
}

func (*MCTo) Type() string {
	return "MCTo"
}

func (m *MCTo) MarshalNMDC(ec *Encoding, buf *bytes.Buffer) error {
	if err := Name(m.To).MarshalNMDC(ec, buf); err != nil {
		return err
	}

	buf.WriteString(" $")
	if err := Name(m.From).MarshalNMDC(ec, buf); err != nil {
		return err
	}

	buf.WriteByte(' ')
	if m.Me {
		buf.WriteString("/me ")
	}
	text := m.Text
	if m.Me && strings.HasPrefix(text, "/me ") {
		text = text[4:]
	}
	if err := String(text).MarshalNMDC(ec, buf); err != nil {
		return err
	}
	return nil
}

func (m *MCTo) UnmarshalNMDC(ec *Encoding, data []byte) error {
	var name Name

	i := bytes.Index(data, []byte(" $"))
	if i < 0 {
		return errors.New("invalid MCTo: no name delimiter")
	} else if err := name.UnmarshalNMDC(ec, data[:i]); err != nil {
		return err
	}
	data = data[i+2:]
	m.To = string(name)

	i = bytes.IndexByte(data, ' ')
	if i < 0 {
		return errors.New("invalid MCTo: no message delimiter")
	} else if err := name.UnmarshalNMDC(ec, data[:i]); err != nil {
		return err
	}
	m.From = string(name)

	var s String
	if err := s.UnmarshalNMDC(ec, data[i+1:]); err != nil {
		return err
	}
	m.Text = string(s)
	m.Me = strings.HasPrefix(m.Text, "/me ")
	if m.Me {
		m.Text = m.Text[4:]
	}
	return nil
}

type UCmdType int

const (
	TypeSeparator      = UCmdType(0)
	TypeRaw            = UCmdType(1)
	TypeRawNickLimited = UCmdType(2)
	TypeErase          = UCmdType(255)
)

type UCmdContext int

const (
	ContextHub      = UCmdContext(1)
	ContextUser     = UCmdContext(2)
	ContextSearch   = UCmdContext(4)
	ContextFileList = UCmdContext(8)
)

type UserCommand struct {
	Typ     UCmdType
	Context UCmdContext
	Path    []string
	Command string
}

func (*UserCommand) Type() string {
	return "UserCommand"
}

func (m *UserCommand) MarshalNMDC(ec *Encoding, buf *bytes.Buffer) error {
	var b [10]byte
	bi := strconv.AppendUint(b[:0], uint64(m.Typ), 10)
	bi = append(bi, ' ')
	bi = strconv.AppendUint(bi, uint64(m.Context), 10)
	if len(m.Path) != 0 {
		bi = append(bi, ' ')
		buf.Write(bi)
		for i, s := range m.Path {
			if i != 0 {
				buf.WriteString("\\")
			}
			if err := String(s).MarshalNMDC(ec, buf); err != nil {
				return err
			}
		}
	} else {
		buf.Write(bi)
	}
	if m.Command == "" {
		return nil
	}
	buf.WriteString(" $")
	if err := String(m.Command).MarshalNMDC(ec, buf); err != nil {
		return err
	}
	return nil
}

func (m *UserCommand) UnmarshalNMDC(ec *Encoding, data []byte) error {
	arr := bytes.SplitN(bytes.TrimSpace(data), []byte(" "), 3)
	if len(arr) < 2 {
		return errors.New("invalid user command")
	}

	t, err := atoiTrim(arr[0])
	if err != nil {
		return fmt.Errorf("invalid type in user command: %q", string(arr[0]))
	}
	m.Typ = UCmdType(t)

	if i := bytes.IndexFunc(arr[1], func(r rune) bool {
		return r < '0' || r > '9'
	}); i >= 0 {
		arr[1] = arr[1][:i]
	}
	c, err := atoiTrim(arr[1])
	if err != nil {
		return fmt.Errorf("invalid context in user command: %q", string(arr[1]))
	}
	m.Context = UCmdContext(c)
	if len(arr) == 2 {
		return nil
	}

	val := arr[2]
	i := bytes.IndexByte(val, '$')
	if i < 1 {
		return fmt.Errorf("invalid raw user command: %q", string(data))
	}
	path := bytes.TrimRight(val[:i], " ")
	sub := bytes.Split(path, []byte("\\"))
	m.Path = make([]string, 0, len(sub))
	for _, p := range sub {
		if len(p) == 0 {
			continue
		}
		var s String
		err := s.UnmarshalNMDC(ec, p)
		if err != nil {
			return errors.New("invalid path user command")
		}
		m.Path = append(m.Path, string(s))
	}

	var s String
	if err := s.UnmarshalNMDC(ec, val[i+1:]); err != nil {
		return err
	}
	m.Command = string(s)
	return nil
}
