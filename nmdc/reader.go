package nmdc

import (
	"bytes"
	"fmt"
	"io"

	"github.com/direct-connect/go-dc/lineproto"
)

type ErrUnexpectedCommand struct {
	Expected string
	Received *RawMessage
}

func (e *ErrUnexpectedCommand) Error() string {
	exp := e.Expected
	if exp == "" {
		exp = "<chat>"
	}
	got := e.Received.Typ
	if got == "" {
		got = "<chat>"
	}
	return fmt.Sprintf("nmdc: expected %q, got %q", exp, got)
}

type ErrProtocolViolation = lineproto.ErrProtocolViolation

type errUnknownEncoding struct {
	text []byte
}

func (e *errUnknownEncoding) Error() string {
	return fmt.Sprintf("nmdc: unknown text encoding: %q", string(e.text))
}

// NewReader creates an NMDC protocol reader
func NewReader(r io.Reader) *Reader {
	return NewReaderOn(lineproto.NewReader(r, Delimiter))
}

func NewReaderOn(r lineproto.LineReader) *Reader {
	return &Reader{
		r:  r,
		ec: DefaultEncoding(),
	}
}

// Reader is safe for concurrent use.
type Reader struct {
	r lineproto.LineReader

	ec *Encoding

	// OnKeepAlive is called when an empty (keep-alive) message is received.
	OnKeepAlive func() error

	// onRawCommand is called each time a message is received.
	// Protocol commands will have a non-nil name, while chat messages will have a nil name.
	// The function may return (false, nil) to ignore the message.
	onRawMessage []func(cmd, args []byte) (bool, error)

	// OnUnknownEncoding is called when a text with non-UTF8 encoding is received.
	// It may either return a new decoder or return an error to fail the decoding.
	OnUnknownEncoding func(text []byte) (TextEncoding, error)

	// OnUnmarshalError is called when a message cannot be parsed.
	// It may either return false, nil to skip the message or true, err to return an error.
	OnUnmarshalError func(text []byte, err error) (bool, error)

	// onMessage is called each time a protocol message is decoded.
	// The function may return (false, nil) to ignore the message.
	onMessage []func(m Message) (bool, error)
}

// OnRawCommand registers a hook that is called each time a message is received.
// Protocol commands will have a non-nil name, while chat messages will have a nil name.
// The function may return (false, nil) to ignore the message.
//
// This method is not concurrent-safe.
func (r *Reader) OnRawMessage(fnc func(cmd, args []byte) (bool, error)) {
	r.onRawMessage = append(r.onRawMessage, fnc)
}

// OnMessage registers a hook that is called each time a protocol message is decoded.
// The function may return (false, nil) to ignore the message.
//
// This method is not concurrent-safe.
func (r *Reader) OnMessage(fnc func(m Message) (bool, error)) {
	r.onMessage = append(r.onMessage, fnc)
}

// Encoding returns current protocol encoding.
func (r *Reader) Encoding() *Encoding {
	return r.ec
}

// SetEncoding sets current protocol encoding.
func (r *Reader) SetEncoding(ec *Encoding) {
	r.ec = ec
}

// ReadMsg reads a single message.
func (r *Reader) ReadMsg() (Message, error) {
	var m Message
	if err := r.readMsgTo(&m); err != nil {
		return nil, err
	}
	return m, nil
}

// ReadMsgTo will read a message to a pointer passed to the function.
// If the message read has a different type, an error will be returned.
func (r *Reader) ReadMsgTo(m Message) error {
	if m == nil {
		panic("nil message to decode")
	}
	return r.readMsgTo(&m)
}

// ReadMsgToAny will read a message to one of the pointers passed to the function.
// If the message read doesn't match any of the types, an error will be returned.
// The method returns a message that was decoded.
func (r *Reader) ReadMsgToAny(arr ...Message) (Message, error) {
	if len(arr) == 0 {
		panic("no messages to decode")
	}
	var m Message
	if err := r.readMsgTo(&m, arr...); err != nil {
		return nil, err
	}
	return m, nil
}

func (r *Reader) readMsgTo(ptr *Message, allowed ...Message) error {
read:
	for {
		line, err := r.r.ReadLine()
		if err != nil {
			return err
		}
		origLine := line
		cmd, args, err := r.ec.splitLine(line)
		if err != nil {
			return err
		}
		if len(cmd)+len(args) == 0 {
			// keep-alive
			if r.OnKeepAlive != nil {
				if err := r.OnKeepAlive(); err != nil {
					return err
				}
			}
			continue // ignore
		}
		for _, fnc := range r.onRawMessage {
			if ok, err := fnc(cmd, args); err != nil {
				return err
			} else if !ok {
				continue read // drop
			}
		}
		var eerr error
		if len(allowed) != 0 {
			var m Message
			m, eerr = r.ec.unmarshalToAny(cmd, args, allowed)
			if err == nil {
				*ptr = m
			}
		} else {
			eerr = r.ec.unmarshalTo(cmd, args, ptr)
		}
		out := *ptr
		if r.OnUnknownEncoding != nil {
			if e, ok := eerr.(*errUnknownEncoding); ok {
				tenc, err := r.OnUnknownEncoding(e.text)
				if err != nil {
					return err
				}
				ec := r.ec
				if tenc == nil {
					// cannot decode, but asked to continue
					// this will replace invalid characters
					args = bytes.Map(func(r rune) rune {
						return r // only need to parse
					}, args)
				} else {
					// switch encoding and decode again
					ec = ec.Clone()
					ec.setTextDecoder(tenc.NewDecoder())
				}
				eerr = out.UnmarshalNMDC(ec, args)
				if eerr == nil && tenc != nil {
					// apply new encoding
					ec.SetTextEncoding(tenc)
					r.ec = ec
				}
			}
		}
		if eerr != nil {
			if r.OnUnmarshalError == nil {
				return eerr
			}
			ok, err := r.OnUnmarshalError(origLine, eerr)
			if !ok {
				continue read // ignore
			}
			return err
		}
		for _, fnc := range r.onMessage {
			if ok, err := fnc(out); err != nil {
				return err
			} else if !ok {
				continue read // drop
			}
		}
		return nil
	}
}
