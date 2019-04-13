package nmdc

import (
	"bytes"
	"errors"
	"io"

	"github.com/direct-connect/go-dc/lineproto"
)

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Writer: lineproto.NewWriter(w),
	}
}

// Writer is not safe for concurrent use.
type Writer struct {
	*lineproto.Writer

	// onMessage is called each time a NMDC protocol message is written.
	// The function may return (false, nil) to skip writing the message.
	onMessage []func(m Message) (bool, error)

	enc  *TextEncoder
	mbuf bytes.Buffer
}

// OnMessage registers a hook that is called each time a NMDC protocol message is written.
// The function may return (false, nil) to skip writing the message.
//
// This method is not concurrent-safe.
func (w *Writer) OnMessage(fnc func(m Message) (bool, error)) {
	w.onMessage = append(w.onMessage, fnc)
}

// Encoder returns current encoder.
func (w *Writer) Encoder() *TextEncoder {
	return w.enc
}

// SetEncoder sets a text encoding used to write messages.
func (w *Writer) SetEncoder(enc *TextEncoder) {
	w.enc = enc
}

// WriteMsg encodes and writes a NMDC protocol message.
func (w *Writer) WriteMsg(m Message) error {
	if err := w.Writer.Err(); err != nil {
		return err
	}
	for _, fnc := range w.onMessage {
		if ok, err := fnc(m); err != nil || !ok {
			return err
		}
	}
	w.mbuf.Reset()
	err := MarshalTo(w.enc, &w.mbuf, m)
	if err != nil {
		return err
	}
	return w.WriteLine(w.mbuf.Bytes())
}

func escapeString(sw *bytes.Buffer, s string) error {
	last := 0
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == 0x00 {
			return errors.New("invalid characters in string")
		} else if escapeCharsString[b] == "" {
			continue
		}
		if last != i {
			sw.WriteString(s[last:i])
		}
		last = i + 1
		sw.WriteString(escapeCharsString[b])
	}
	if last != len(s) {
		sw.WriteString(s[last:])
	}
	return nil
}

func escapeName(sw *bytes.Buffer, s string) error {
	last := 0
	for i := 0; i < len(s); i++ {
		b := s[i]
		if invalidCharsNameI[b] {
			return errors.New("invalid characters in name")
		} else if escapeCharsName[b] == "" {
			continue
		}
		if last != i {
			sw.WriteString(s[last:i])
		}
		last = i + 1
		sw.WriteString(escapeCharsName[b])
	}
	if last != len(s) {
		sw.WriteString(s[last:])
	}
	return nil
}
