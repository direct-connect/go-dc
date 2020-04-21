package nmdc

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
)

// Delimiter of the NMDC protocol.
const Delimiter = '|'

var (
	messages = make(map[string]reflect.Type)
)

// Message is an interface for all NMDC protocol messages.
type Message interface {
	// Type returns a NMDC command type name without '$' prefix.
	// Chat messages are a special case and returns empty type name.
	Type() string
	// MarshalNMDC encodes NMDC protocol message using provided text encoding into a buffer.
	// Message should only encode it's payload without a command name or '|' delimiter.
	// If encoder is nil, UTF-8 encoding is assumed.
	MarshalNMDC(ec *Encoding, buf *bytes.Buffer) error
	// UnmarshalNMDC decodes NMDC protocol message using provided text encoding.
	// Buffer will only contain message payload without a command name or '|' delimiter.
	// If decoder is nil, UTF-8 encoding is assumed.
	UnmarshalNMDC(ec *Encoding, data []byte) error
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
func (m *RawMessage) MarshalNMDC(_ *Encoding, buf *bytes.Buffer) error {
	buf.Write(m.Data)
	return nil
}

// Type implements Message.
func (m *RawMessage) UnmarshalNMDC(_ *Encoding, data []byte) error {
	m.Data = make([]byte, len(data))
	copy(m.Data, data)
	return nil
}

// MarshalTo encodes NMDC message into a buffer. It will write a command name,
// a payload and '|' delimiter.
//
// Deprecated: use Encoding.MarshalTo
func MarshalTo(enc *TextEncoder, buf *bytes.Buffer, m Message) error {
	ec := DefaultEncoding()
	if enc != nil {
		ec = ec.Clone()
		ec.setTextEncoder(enc)
	}
	return ec.MarshalTo(buf, m)
}

// Marshal encodes NMDC message. The resulting slice will contain a command name,
// a payload and '|' delimiter.
//
// Deprecated: use Encoding.Marshal
func Marshal(enc *TextEncoder, m Message) ([]byte, error) {
	ec := DefaultEncoding()
	if enc != nil {
		ec = ec.Clone()
		ec.setTextEncoder(enc)
	}
	return ec.Marshal(m)
}

// Unmarshal decodes a single NMDC message from the buffer.
//
// Deprecated: use Encoding.Unmarshal
func Unmarshal(dec *TextDecoder, data []byte) (Message, error) {
	ec := DefaultEncoding()
	if dec != nil {
		ec = ec.Clone()
		ec.setTextDecoder(dec)
	}
	return ec.Unmarshal(data)
}
