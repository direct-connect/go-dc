package nmdc

import (
	"bytes"
	"io"
	"sync"

	"github.com/direct-connect/go-dc/lineproto"
)

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		AsyncWriter: lineproto.NewAsyncWriter(w),
	}
}

// Writer is safe for concurrent use.
type Writer struct {
	*lineproto.AsyncWriter

	mu   sync.Mutex
	mbuf bytes.Buffer
	enc  *TextEncoder
}

// Encoder returns current encoder.
func (w *Writer) Encoder() *TextEncoder {
	w.mu.Lock()
	enc := w.enc
	w.mu.Unlock()
	return enc
}

// SetEncoder sets a text encoding used to write messages.
func (w *Writer) SetEncoder(enc *TextEncoder) {
	w.mu.Lock()
	w.enc = enc
	w.mu.Unlock()
}

// WriteMsg encodes and writes a NMDC protocol message.
func (w *Writer) WriteMsg(m Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.Writer.Err(); err != nil {
		return err
	}
	w.mbuf.Reset()
	err := MarshalTo(w.enc, &w.mbuf, m)
	if err != nil {
		return err
	}
	return w.WriteLine(w.mbuf.Bytes())
}

// WriteMsgAsync encodes and writes a NMDC protocol message asynchronously.
// The message won't be sent immediately, instead, it will be batched with other similar
// messages.
func (w *Writer) WriteMsgAsync(m Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.Writer.Err(); err != nil {
		return err
	}
	w.mbuf.Reset()
	err := MarshalTo(w.enc, &w.mbuf, m)
	if err != nil {
		return err
	}
	return w.WriteLineAsync(w.mbuf.Bytes())
}
