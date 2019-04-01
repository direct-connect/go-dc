package nmdc

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"

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

	// onMessage is called each time a NMDC protocol message is written.
	// The function may return (false, nil) to skip writing the message.
	onMessage []func(m Message) (bool, error)

	encNolock atomic.Value // synced with enc

	mu   sync.Mutex
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
	v, _ := w.encNolock.Load().(*TextEncoder)
	return v
}

// SetEncoder sets a text encoding used to write messages.
func (w *Writer) SetEncoder(enc *TextEncoder) {
	w.mu.Lock()
	w.enc = enc
	w.encNolock.Store(enc)
	w.mu.Unlock()
}

// WriteMsg encodes and writes a NMDC protocol message.
func (w *Writer) WriteMsg(m Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
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

// WriteMsgAsync encodes and writes a NMDC protocol message asynchronously.
// The message won't be sent immediately, instead, it will be batched with other similar
// messages.
func (w *Writer) WriteMsgAsync(m Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
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
	return w.WriteLineAsync(w.mbuf.Bytes())
}
