package nmdc

import (
	"bytes"
	"io"

	"github.com/direct-connect/go-dc/lineproto"
)

// NewWriter creates a new NMDC protocol writer with a default buffer size.
func NewWriter(w io.Writer) *Writer {
	return NewWriterSize(w, 0)
}

// NewWriterSize creates a new NMDC protocol writer with a specified buffer size.
func NewWriterSize(w io.Writer, buf int) *Writer {
	return NewWriterOn(lineproto.NewWriterSize(w, buf))
}

// NewWriter creates a new NMDC protocol writer with a default buffer size.
func NewWriterOn(w lineproto.LineWriter) *Writer {
	return &Writer{
		w: w,
	}
}

// Writer is protocol message writer for NMDC protocol. It's not safe for a concurrent use.
type Writer struct {
	w lineproto.LineWriter

	ec *Encoding

	// onMessage is called each time a NMDC protocol message is written.
	// The function may return (false, nil) to skip writing the message.
	onMessage []func(m Message) (bool, error)

	mbuf bytes.Buffer
}

func (w *Writer) Reset(lw lineproto.LineWriter) {
	w.mbuf.Reset()
	w.w = lw
}

// OnMessage registers a hook that is called each time a NMDC protocol message is written.
// The function may return (false, nil) to skip writing the message.
//
// This method is not concurrent-safe.
func (w *Writer) OnMessage(fnc func(m Message) (bool, error)) {
	w.onMessage = append(w.onMessage, fnc)
}

// Encoding returns current protocol encoding.
func (w *Writer) Encoding() *Encoding {
	return w.ec
}

// SetEncoding sets current protocol encoding.
func (w *Writer) SetEncoding(ec *Encoding) {
	w.ec = ec
}

// WriteKeepAlive writes an empty (keep alive) message.
// It is caller's responsibility to flush the writer.
func (w *Writer) WriteKeepAlive() error {
	return w.w.WriteLine([]byte{Delimiter})
}

// WriteMsg encodes and writes a NMDC protocol message.
func (w *Writer) WriteMsg(msg ...Message) error {
	enc := w.ec
loop:
	for _, m := range msg {
		for _, fnc := range w.onMessage {
			if ok, err := fnc(m); err != nil {
				return err
			} else if !ok {
				continue loop // drop
			}
		}
		w.mbuf.Reset()
		err := enc.MarshalTo(&w.mbuf, m)
		if err != nil {
			return err
		}
		if err := w.w.WriteLine(w.mbuf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}
