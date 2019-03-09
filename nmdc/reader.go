package nmdc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/direct-connect/go-dc/lineproto"
)

const (
	maxName    = 256
	maxCmdName = 32
)

var (
	errExpectedCommand = errors.New("nmdc: expected command, got chat message")
	errExpectedChat    = errors.New("nmdc: chat message, got command")
)

type ErrUnexpectedCommand struct {
	Expected string
	Received *RawMessage
}

func (e *ErrUnexpectedCommand) Error() string {
	return fmt.Sprintf("nmdc: expected %q, got %q", e.Expected, e.Received.Typ)
}

type ErrProtocolViolation = lineproto.ErrProtocolViolation

type errUnknownEncoding struct {
	text []byte
}

func (e *errUnknownEncoding) Error() string {
	return fmt.Sprintf("nmdc: unknown text encoding: %q", string(e.text))
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		Reader:     lineproto.NewReader(r, '|'),
		maxCmdName: maxCmdName,
	}
}

// Reader is safe for concurrent use.
type Reader struct {
	*lineproto.Reader

	maxCmdName int

	mu sync.Mutex
	// dec is the current decoder for the text values.
	// It converts connection encoding to UTF8. Nil value means that connection uses UTF8.
	dec *TextDecoder

	// OnKeepAlive is called when an empty (keep-alive) message is received.
	OnKeepAlive func() error

	// OnRawCommand is called each time a message is received.
	// Protocol commands will have a non-nil name, while chat messages will have a nil name.
	// The function may return (false, nil) to ignore the message.
	OnRawMessage func(cmd, args []byte) (bool, error)

	// OnUnknownEncoding is called when a text with non-UTF8 encoding is received.
	// It may either return a new decoder or return an error to fail the decoding.
	OnUnknownEncoding func(text []byte) (*TextDecoder, error)

	// OnMessage is called each time a protocol message is decoded.
	// The function may return (false, nil) to ignore the message.
	OnMessage func(m Message) (bool, error)
}

// SetMaxCmdName sets a maximal length of the protocol command name in bytes.
func (r *Reader) SetMaxCmdName(n int) {
	r.maxCmdName = n
}

// SetDecoder sets a text decoder for the connection.
func (r *Reader) SetDecoder(dec *TextDecoder) {
	r.mu.Lock()
	r.dec = dec
	r.mu.Unlock()
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

func (r *Reader) readMsgTo(ptr *Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for {
		line, err := r.ReadLine()
		if err != nil {
			return err
		}
		if bytes.ContainsAny(line, "\x00") {
			return &ErrProtocolViolation{
				Err: errors.New("message should not contain null characters"),
			}
		}
		line = bytes.TrimSuffix(line, []byte("|"))
		if len(line) == 0 {
			// keep-alive
			if r.OnKeepAlive != nil {
				if err := r.OnKeepAlive(); err != nil {
					return err
				}
			}
			continue // keep alive, ignore
		}
		var (
			out  = *ptr
			cmd  []byte
			args []byte
		)
		if line[0] == '$' {
			line = line[1:]
			// protocol command
			cmd, args = line, nil // only name
			if i := bytes.Index(line, []byte(" ")); i >= 0 {
				cmd, args = line[:i], line[i+1:] // name and args
			}
			if r.OnRawMessage != nil {
				if ok, err := r.OnRawMessage(cmd, args); err != nil {
					return err
				} else if !ok {
					continue // drop
				}
			}
			if len(cmd) == 0 {
				return &ErrProtocolViolation{
					Err: errors.New("command name is empty"),
				}
			} else if len(cmd) > r.maxCmdName {
				return &ErrProtocolViolation{
					Err: errors.New("command name is too long"),
				}
			} else if !isASCII(cmd) {
				return &ErrProtocolViolation{
					Err: fmt.Errorf("command name should be in acsii: %q", string(cmd)),
				}
			}
			if out == nil {
				// detect type by command name
				out = NewMessage(string(cmd))
				*ptr = out
			} else if _, ok := out.(*ChatMessage); ok {
				return errExpectedCommand
			} else if typ := string(cmd); out.Type() != typ {
				return &ErrUnexpectedCommand{
					Expected: out.Type(),
					Received: &RawMessage{
						Typ: typ, Data: args,
					},
				}
			}
		} else {
			// chat message
			cmd, args = nil, line
			if r.OnRawMessage != nil {
				if ok, err := r.OnRawMessage(cmd, args); err != nil {
					return err
				} else if !ok {
					continue // drop
				}
			}
			if out == nil {
				out = &ChatMessage{}
				*ptr = out
			} else if _, ok := out.(*ChatMessage); !ok {
				return errExpectedChat
			}
		}
		err = out.UnmarshalNMDC(r.dec, args)
		if r.OnUnknownEncoding != nil {
			if e, ok := err.(*errUnknownEncoding); ok {
				dec, err := r.OnUnknownEncoding(e.text)
				if err != nil {
					return err
				} else if dec == nil {
					return e
				}
				r.dec = dec
				err = out.UnmarshalNMDC(r.dec, args)
			}
		}
		if err != nil {
			return err
		}
		if r.OnMessage != nil {
			if ok, err := r.OnMessage(out); err != nil {
				return err
			} else if !ok {
				continue // drop
			}
		}
		return nil
	}
}

func isASCII(p []byte) bool {
	for _, b := range p {
		if b == '/' || b == '-' || b == '_' || b == '.' || b == ':' {
			continue
		}
		if b < '0' || b > 'z' {
			return false
		}
		if b >= 'a' && b <= 'z' {
			continue
		}
		if b >= 'A' && b <= 'Z' {
			continue
		}
		if b >= '0' && b <= '9' {
			continue
		}
		return false
	}
	return true
}
