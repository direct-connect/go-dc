package adc

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strconv"
)

var (
	messages = make(map[MsgType]reflect.Type)
)

func init() {
	RegisterMessage(Supported{})
	RegisterMessage(Status{})
	RegisterMessage(ZOn{})
	RegisterMessage(ZOff{})
}

type Message interface {
	Cmd() MsgType
}

func RegisterMessage(m Message) {
	name := m.Cmd()
	if _, ok := messages[name]; ok {
		panic(fmt.Errorf("%q already registered", name.String()))
	}
	rt := reflect.TypeOf(m)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	messages[name] = rt
}

func UnmarshalMessage(name MsgType, data []byte) (Message, error) {
	rt, ok := messages[name]
	if !ok {
		r := &RawMessage{Type: name}
		if err := r.UnmarshalADC(data); err != nil {
			return nil, err
		}
		return r, nil
	}
	rv := reflect.New(rt)
	if err := Unmarshal(data, rv.Interface()); err != nil {
		return nil, err
	}
	return rv.Elem().Interface().(Message), nil
}

var (
	_ Message     = (*RawMessage)(nil)
	_ Marshaler   = (*RawMessage)(nil)
	_ Unmarshaler = (*RawMessage)(nil)
)

type RawMessage struct {
	Type MsgType
	Data []byte
}

func (m *RawMessage) Cmd() MsgType {
	return m.Type
}

func (m *RawMessage) Decode() (Message, error) {
	return UnmarshalMessage(m.Type, m.Data)
}

func (m *RawMessage) MarshalADC(buf *bytes.Buffer) error {
	buf.Write(m.Data)
	return nil
}

func (m *RawMessage) UnmarshalADC(data []byte) error {
	m.Data = append([]byte{}, data...)
	return nil
}

var (
	_ Marshaler   = NoArgs{}
	_ Unmarshaler = NoArgs{}
)

type NoArgs struct{}

func (NoArgs) MarshalADC(buf *bytes.Buffer) error {
	return nil
}

func (NoArgs) UnmarshalADC(data []byte) error {
	// TODO: check size
	return nil
}

var (
	_ Marshaler   = Fields{}
	_ Unmarshaler = (*Fields)(nil)
)

type Field struct {
	Tag   [2]byte
	Value string
}

func (f Field) String() string {
	return string(f.Tag[:]) + string(f.Value)
}

type Fields []Field

func (f Fields) MarshalADC(buf *bytes.Buffer) error {
	for i, f := range f {
		if i != 0 {
			buf.WriteByte(' ')
		}
		buf.Write(f.Tag[:])
		if err := String(f.Value).MarshalADC(buf); err != nil {
			return err
		}
	}
	return nil
}

func (f *Fields) UnmarshalADC(data []byte) error {
	*f = (*f)[:0]
	if len(data) == 0 {
		return nil
	}
	mp := *f
	sub := bytes.Split(data, []byte(" "))
	for _, v := range sub {
		if len(v) < 2 {
			return fmt.Errorf("invalid field: %q", string(v))
		}
		k := [2]byte{v[0], v[1]}
		v = v[2:]
		var s String
		if err := s.UnmarshalADC(v); err != nil {
			return fmt.Errorf("cannot unmarshal field %s: %v", string(k[:]), err)
		}
		mp = append(mp, Field{Tag: k, Value: string(v)})
	}
	*f = mp
	return nil
}

var (
	_ Message     = Supported{}
	_ Marshaler   = Supported{}
	_ Unmarshaler = (*Supported)(nil)
)

type Supported struct {
	Features ModFeatures
}

func (Supported) Cmd() MsgType {
	return MsgType{'S', 'U', 'P'}
}

func (m Supported) MarshalADC(buf *bytes.Buffer) error {
	return m.Features.MarshalADC(buf)
}

func (m *Supported) UnmarshalADC(data []byte) error {
	return m.Features.UnmarshalADC(data)
}

type Severity int

const (
	Success     = Severity(0)
	Recoverable = Severity(1)
	Fatal       = Severity(2)
)

var (
	_ Message     = Status{}
	_ Marshaler   = Status{}
	_ Unmarshaler = (*Status)(nil)
)

type Status struct {
	Sev  Severity
	Code int
	Msg  string
}

func (Status) Cmd() MsgType {
	return MsgType{'S', 'T', 'A'}
}

func (st Status) Ok() bool {
	return st.Sev == Success
}

func (st Status) Recoverable() bool {
	return st.Ok() || st.Sev == Recoverable
}

func (st Status) Err() error {
	if !st.Ok() {
		if st.Code == 51 {
			return os.ErrNotExist
		}
		return Error{st}
	}
	return nil
}

func (st Status) MarshalADC(buf *bytes.Buffer) error {
	buf.WriteString(fmt.Sprintf("%d%02d ", int(st.Sev), st.Code))
	buf.Write(escape(st.Msg))
	return nil
}

func (st *Status) UnmarshalADC(s []byte) error {
	sub := bytes.SplitN(s, []byte(" "), 2)
	code, err := strconv.Atoi(string(sub[0]))
	if err != nil {
		return fmt.Errorf("wrong status code: %v", err)
	}
	st.Code = code % 100
	st.Sev = Severity(code / 100)
	st.Msg = ""
	if len(sub) > 1 {
		st.Msg = unescape(sub[1])
	}
	return nil
}

var (
	_ Message     = ZOn{}
	_ Marshaler   = ZOn{}
	_ Unmarshaler = (*ZOn)(nil)
)

type ZOn struct {
	NoArgs
}

func (ZOn) Cmd() MsgType {
	return MsgType{'Z', 'O', 'N'}
}

var (
	_ Message     = ZOff{}
	_ Marshaler   = ZOff{}
	_ Unmarshaler = (*ZOff)(nil)
)

type ZOff struct {
	NoArgs
}

func (ZOff) Cmd() MsgType {
	return MsgType{'Z', 'O', 'F'}
}
