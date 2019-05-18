package nmdc

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"

	"golang.org/x/text/encoding"
)

const lineDelim = '|'

var (
	messages = make(map[string]reflect.Type)
)

// these two aliases allows package users to omit x/text import
type (
	TextEncoder = encoding.Encoder
	TextDecoder = encoding.Decoder
)

// Message is an interface for all NMDC protocol messages.
type Message interface {
	// Type returns a NMDC command type name without '$' prefix.
	// Chat messages are a special case and returns empty type name.
	Type() string
	// MarshalNMDC encodes NMDC protocol message using provided text encoding into a buffer.
	// Message should only encode it's payload without a command name or '|' delimiter.
	// If encoder is nil, UTF-8 encoding is assumed.
	MarshalNMDC(enc *TextEncoder, buf *bytes.Buffer) error
	// UnmarshalNMDC decodes NMDC protocol message using provided text encoding.
	// Buffer will only contain message payload without a command name or '|' delimiter.
	// If decoder is nil, UTF-8 encoding is assumed.
	UnmarshalNMDC(dec *TextDecoder, data []byte) error
}

// RegisterMessage registers a new protocol message type. It will be associated with a
// name returned by Type method. Messages registered this way will be automatically decoded
// by the Reader.
func RegisterMessage(m Message) {
	typ := m.Type()
	if _, ok := messages[typ]; ok {
		panic(fmt.Errorf("message type %q is already registered", typ))
	}
	rt := reflect.TypeOf(m)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	messages[typ] = rt
}

// NewMessage creates a new message by type name. If a command type is unknown,
// RawMessage will be returned.
//
// See RegisterMessage for more details.
func NewMessage(typ string) Message {
	rt, ok := messages[typ]
	if !ok {
		return &RawMessage{Typ: typ}
	}
	return reflect.New(rt).Interface().(Message)
}

// RegisteredTypes list all registered message types.
func RegisteredTypes() []string {
	arr := make([]string, 0, len(messages))
	for typ := range messages {
		arr = append(arr, typ)
	}
	sort.Strings(arr)
	return arr
}

// IsRegistered check if a message type is registered.
func IsRegistered(typ string) bool {
	_, ok := messages[typ]
	return ok
}

// IsRegisteredBytes is like IsRegistered but accepts a byte slice.
func IsRegisteredBytes(typ []byte) bool {
	_, ok := messages[string(typ)]
	return ok
}

var _ Message = (*RawMessage)(nil)

// RawMessage is a raw NMDC message in the connection encoding.
type RawMessage struct {
	Typ  string
	Data []byte
}

// Type implements Message.
func (m *RawMessage) Type() string {
	return m.Typ
}

// Type implements Message.
func (m *RawMessage) MarshalNMDC(_ *TextEncoder, buf *bytes.Buffer) error {
	buf.Write(m.Data)
	return nil
}

// Type implements Message.
func (m *RawMessage) UnmarshalNMDC(_ *TextDecoder, data []byte) error {
	m.Data = make([]byte, len(data))
	copy(m.Data, data)
	return nil
}

// MarshalTo encodes NMDC message into a buffer. It will write a command name,
// a payload and '|' delimiter.
func MarshalTo(enc *TextEncoder, buf *bytes.Buffer, m Message) error {
	if typ := m.Type(); typ != "" {
		buf.Grow(1 + len(typ) + 2)
		buf.WriteByte('$')
		buf.WriteString(typ)
		buf.WriteByte(' ')
	}
	n := buf.Len()
	if err := m.MarshalNMDC(enc, buf); err != nil {
		return err
	}
	if n == buf.Len() {
		// no payload
		buf.Bytes()[n-1] = lineDelim // ' ' -> '|'
		return nil
	}
	buf.WriteByte(lineDelim)
	return nil
}

// Marshal encodes NMDC message. The resulting slice will contain a command name,
// a payload and '|' delimiter.
func Marshal(enc *TextEncoder, m Message) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := MarshalTo(enc, buf, m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal decodes a single NMDC message from the buffer.
func Unmarshal(dec *TextDecoder, data []byte) (Message, error) {
	// TODO: implement separate decoder?
	r := NewReader(bytes.NewReader(data))
	r.SetDecoder(dec)
	m, err := r.ReadMsg()
	if err == io.EOF {
		return nil, io.ErrUnexpectedEOF
	}
	return m, err
}
