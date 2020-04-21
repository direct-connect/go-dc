package nmdc

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/text/language"
	"sync/atomic"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
)

const (
	maxName    = 256
	maxCmdName = 32
)

// these two aliases allows package users to omit x/text import
type (
	TextEncoding = encoding.Encoding
	TextEncoder  = encoding.Encoder
	TextDecoder  = encoding.Decoder
)

func TextEncodingByName(name string) (TextEncoding, error) {
	enc, err := htmlindex.Get(name)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

func encodingForLanguage(lang string) (TextEncoding, bool) {
	if lang == "" {
		return nil, false
	}
	l, err := language.Parse(lang)
	if err != nil {
		return nil, false
	}
	name := htmlindex.LanguageDefault(l)
	if name == "" {
		return nil, false
	}
	enc, err := htmlindex.Get(name)
	if err != nil {
		return nil, false
	}
	return enc, true
}

func NewEncoding() *Encoding {
	return &Encoding{
		MaxCmdName:  maxCmdName,
		MaxUserName: maxName,
	}
}

var defEncoding = NewEncoding()

func DefaultEncoding() *Encoding {
	return defEncoding
}

type Encoding struct {
	MaxUserName int  // maximal length of the username in bytes
	MaxCmdName  int  // maximal length of the protocol command name in bytes
	Raw         bool // do not check text values

	// ecg is the current encoding for the text values.
	ecg atomic.Value // TextEncoding

	// enc is the current encoder for the text values.
	// It converts text encoding from UTF8. Nil value means that the text must stay in UTF8.
	enc atomic.Value // *TextEncoder

	// dec is the current decoder for the text values.
	// It converts text encoding to UTF8. Nil value means that the text is already UTF8.
	dec atomic.Value // *TextDecoder
}

func (ec *Encoding) Equal(ec2 *Encoding) bool {
	if ec == ec2 {
		return true
	}
	if ec.MaxUserName != ec2.MaxUserName || ec.MaxCmdName != ec2.MaxCmdName || ec.Raw != ec2.Raw {
		return false
	}
	return ec.TextEncoding() == ec2.TextEncoding()
}

func (ec *Encoding) Clone() *Encoding {
	if ec == nil && defEncoding != nil {
		ec = defEncoding
	}
	ec2 := &Encoding{
		MaxUserName: ec.MaxUserName,
		MaxCmdName:  ec.MaxCmdName,
		Raw:         ec.Raw,
	}
	ec2.setTextDecoder(ec.TextDecoder())
	ec2.setTextEncoder(ec.TextEncoder())
	return ec2
}

// SetTextEncoding sets a text encoding (both encoder and decoder) for the connection.
func (ec *Encoding) SetTextEncoding(enc TextEncoding) {
	if ec == defEncoding {
		panic("should not change text encoding globally")
	}
	ec.ecg.Store(enc)
	ec.setTextDecoder(enc.NewDecoder())
	ec.setTextEncoder(enc.NewEncoder())
}

func (ec *Encoding) TextEncoding() TextEncoding {
	if ec == nil {
		return nil
	}
	ecg, _ := ec.ecg.Load().(TextEncoding)
	return ecg
}

// TextDecoder returns current text decoder.
func (ec *Encoding) TextDecoder() *TextDecoder {
	if ec == nil {
		return nil
	}
	dec, _ := ec.dec.Load().(*TextDecoder)
	return dec
}

// setTextDecoder sets a text decoder for the connection.
func (ec *Encoding) setTextDecoder(dec *TextDecoder) {
	if ec == defEncoding {
		panic("should not change text encoding globally")
	}
	ec.dec.Store(dec)
}

// TextEncoder returns current text encoder.
func (ec *Encoding) TextEncoder() *TextEncoder {
	if ec == nil {
		return nil
	}
	enc, _ := ec.enc.Load().(*TextEncoder)
	return enc
}

// setTextEncoder sets a text encoder for the connection.
func (ec *Encoding) setTextEncoder(enc *TextEncoder) {
	if ec == defEncoding {
		panic("should not change text encoding globally")
	}
	ec.enc.Store(enc)
}

func (ec *Encoding) DecodeStringBytes(data []byte) ([]byte, error) {
	data = UnescapeBytes(data)
	if ec.Raw {
		return data, nil
	}
	// always allow valid UTF-8, even if text decoder is set
	if utf8.Valid(data) {
		return data, nil
	}
	// invalid UTF-8 - must decode
	dec := ec.TextDecoder()
	if dec == nil {
		return nil, &errUnknownEncoding{text: data}
	}
	data, err := dec.Bytes(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (ec *Encoding) ValidateCmdName(cmd []byte) error {
	if len(cmd) == 0 {
		return &ErrProtocolViolation{
			Err: errors.New("command name is empty"),
		}
	}
	if len(cmd) > ec.MaxCmdName {
		return &ErrProtocolViolation{
			Err: errors.New("command name is too long"),
		}
	}
	if !ec.Raw && !isASCII(cmd) {
		return &ErrProtocolViolation{
			Err: fmt.Errorf("command name should be in acsii: %q", string(cmd)),
		}
	}
	return nil
}

func (ec *Encoding) ValidateLine(line []byte) error {
	if n := len(line); n == 0 || line[n-1] != Delimiter {
		return &ErrProtocolViolation{
			Err: errors.New("no message delimiter"),
		}
	}
	if bytes.IndexByte(line, 0x00) >= 0 {
		return &ErrProtocolViolation{
			Err: errors.New("message should not contain null characters"),
		}
	}
	return nil
}

// MarshalTo encodes NMDC message into a buffer. It will write a command name,
// a payload and '|' delimiter.
func (ec *Encoding) MarshalTo(buf *bytes.Buffer, m Message) error {
	if typ := m.Type(); typ != "" {
		buf.Grow(1 + len(typ) + 2)
		buf.WriteByte('$')
		buf.WriteString(typ)
		buf.WriteByte(' ')
	}
	n := buf.Len()
	if err := m.MarshalNMDC(ec, buf); err != nil {
		return err
	}
	if n == buf.Len() {
		// no payload
		buf.Bytes()[n-1] = Delimiter // ' ' -> '|'
		return nil
	}
	buf.WriteByte(Delimiter)
	return nil
}

// Marshal encodes NMDC message. The resulting slice will contain a command name,
// a payload and '|' delimiter.
func (ec *Encoding) Marshal(m Message) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := ec.MarshalTo(buf, m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (ec *Encoding) splitLine(line []byte) (cmd, args []byte, _ error) {
	if err := ec.ValidateLine(line); err != nil {
		return nil, nil, err
	}
	line = line[:len(line)-1] // trim delimiter
	if len(line) == 0 {
		// keep-alive
		return nil, nil, nil
	}
	if line[0] != '$' {
		// chat message
		return nil, line, nil
	}
	line = line[1:]
	// protocol command
	cmd, args = line, nil // only name as the default
	if i := bytes.IndexByte(line, ' '); i >= 0 {
		cmd, args = line[:i], line[i+1:] // name and args
	}
	err := ec.ValidateCmdName(cmd)
	return cmd, args, err
}

func (ec *Encoding) Unmarshal(line []byte) (Message, error) {
	cmd, args, err := ec.splitLine(line)
	if err != nil {
		return nil, err
	}
	if len(cmd)+len(args) == 0 {
		return &KeepAlive{}, nil
	}
	return ec.unmarshal(cmd, args)
}

func (ec *Encoding) unmarshal(cmd, args []byte) (Message, error) {
	if len(cmd) == 0 {
		// chat message
		var m ChatMessage
		err := m.UnmarshalNMDC(ec, args)
		return &m, err
	}
	// other commands
	m := NewMessage(string(cmd))
	err := m.UnmarshalNMDC(ec, args)
	return m, err
}

func (ec *Encoding) unmarshalTo(cmd, args []byte, ptr *Message) error {
	var out = *ptr
	switch out := out.(type) {
	case nil:
		m, err := ec.unmarshal(cmd, args)
		*ptr = m
		return err
	case *RawMessage:
		out.Typ = string(cmd)
		return out.UnmarshalNMDC(ec, args)
	default:
		if typ, exp := string(cmd), out.Type(); typ != exp {
			return &ErrUnexpectedCommand{
				Expected: exp,
				Received: &RawMessage{
					Typ:  typ,
					Data: args,
				},
			}
		}
		return out.UnmarshalNMDC(ec, args)
	}
}

func (ec *Encoding) unmarshalToAny(cmd, args []byte, allowed []Message) (Message, error) {
	if len(allowed) == 0 {
		panic("call unmarshalTo instead")
	}
	typ := string(cmd)
	var out Message
	for _, m := range allowed {
		if raw, ok := m.(*RawMessage); ok {
			raw.Typ = typ
			out = raw
			break
		} else if m.Type() == typ {
			out = m
			break
		}
	}
	if out == nil {
		return nil, &ErrUnexpectedCommand{
			Expected: allowed[0].Type(),
			Received: &RawMessage{
				Typ:  typ,
				Data: args,
			},
		}
	}
	if err := out.UnmarshalNMDC(ec, args); err != nil {
		return nil, err
	}
	return out, nil
}
