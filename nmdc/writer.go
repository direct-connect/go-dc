package nmdc

import (
	"bytes"
	"compress/zlib"
	"io"
	"sync/atomic"

	"github.com/direct-connect/go-dc/lineproto"
)

// NewWriter creates a new NMDC protocol writer with a default buffer size.
func NewWriter(w io.Writer) *Writer {
	return NewWriterSize(w, 0)
}

// NewWriterSize creates a new NMDC protocol writer with a specified buffer size.
func NewWriterSize(w io.Writer, buf int) *Writer {
	return &Writer{
		Writer: lineproto.NewWriterSize(w, buf),
	}
}

// Writer is protocol message writer for NMDC protocol. It's not safe for a concurrent use.
type Writer struct {
	*lineproto.Writer

	enc atomic.Value // *TextEncoder

	// onMessage is called each time a NMDC protocol message is written.
	// The function may return (false, nil) to skip writing the message.
	onMessage []func(m Message) (bool, error)

	mbuf bytes.Buffer
}

// OnMessage registers a hook that is called each time a NMDC protocol message is written.
// The function may return (false, nil) to skip writing the message.
//
// This method is not concurrent-safe.
func (w *Writer) OnMessage(fnc func(m Message) (bool, error)) {
	w.onMessage = append(w.onMessage, fnc)
}

// Encoder returns current text encoder.
func (w *Writer) Encoder() *TextEncoder {
	enc, _ := w.enc.Load().(*TextEncoder)
	return enc
}

// SetEncoder sets a text encoding used to write messages.
func (w *Writer) SetEncoder(enc *TextEncoder) {
	w.enc.Store(enc)
}

// WriteKeepAlive writes an empty (keep alive) message.
// It is caller's responsibility to flush the writer.
func (w *Writer) WriteKeepAlive() error {
	return w.WriteLine([]byte{Delimiter})
}

// WriteMsg encodes and writes a NMDC protocol message.
func (w *Writer) WriteMsg(msg ...Message) error {
	if err := w.Writer.Err(); err != nil {
		return err
	}
	enc := w.Encoder()
	for _, m := range msg {
		for _, fnc := range w.onMessage {
			if ok, err := fnc(m); err != nil || !ok {
				return err
			}
		}
		w.mbuf.Reset()
		err := MarshalTo(enc, &w.mbuf, m)
		if err != nil {
			return err
		}
		if err := w.WriteLine(w.mbuf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

// ZOn enables compression on this writer.
func (w *Writer) ZOn() error {
	return w.ZOnLevel(zlib.DefaultCompression)
}

// ZOnLevel enables compression with a given level on this writer.
func (w *Writer) ZOnLevel(lvl int) error {
	if err := w.WriteLine([]byte("$" + zonName + "|")); err != nil {
		return err
	}
	// flushes
	return w.EnableZlibLevel(lvl)
}
