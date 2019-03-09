package lineproto

import (
	"bufio"
	"errors"
	"io"
	"sync"
	"time"
)

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: w, bw: bufio.NewWriter(w),
	}
}

// Writer is safe for concurrent use.
type Writer struct {
	// OnLine is called each time a raw protocol message is written.
	// The function may return (false, nil) to skip writing the message.
	OnLine func(line []byte) (bool, error)

	mu  sync.Mutex
	w   io.Writer
	bw  *bufio.Writer
	err error
	lvl int
}

func (w *Writer) Err() error {
	w.mu.Lock()
	err := w.err
	w.mu.Unlock()
	return err
}

func (w *Writer) flush() error {
	err := w.bw.Flush()
	if w.err != nil {
		w.err = err
	}
	return err
}

// Flush forces all buffered writes to be flushed. Partial batch data will be flushed as well.
func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flush()
}

// writeLine writes a single protocol message.
func (w *Writer) writeLine(data []byte) error {
	if w.err != nil {
		return w.err
	}
	if w.OnLine != nil {
		if ok, err := w.OnLine(data); err != nil {
			w.err = err
			return err
		} else if !ok {
			return nil
		}
	}
	if w.lvl != 0 {
		// someone will flush for us
		_, err := w.bw.Write(data)
		if err != nil {
			w.err = err
		}
		return err
	}
	if w.bw.Size() != 0 {
		// buffer not empty - write through it
		_, err := w.bw.Write(data)
		if err != nil {
			w.err = err
			return err
		}
		return w.flush()
	}
	// empty buffer - write directly
	_, err := w.w.Write(data)
	if w.err != nil {
		w.err = err
	}
	return err
}

// WriteLine writes a single protocol message.
func (w *Writer) WriteLine(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writeLine(data)
}

// StartBatch starts a batch of messages. Caller should call EndBatch to flush the buffer.
func (w *Writer) StartBatch() error {
	w.mu.Lock()
	w.lvl++
	err := w.err
	w.mu.Unlock()
	return err
}

// EndBatch flushes a batch of messages. If force is set, the batch will be flushed
// immediately instead of interleaving with other batches.
func (w *Writer) EndBatch(force bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lvl--
	if w.lvl < 0 {
		return errors.New("unpaired EndBatch")
	}
	if !force && w.lvl > 0 {
		// someone will flush for us
		return nil
	}
	return w.flush()
}

func NewAsyncWriter(w io.Writer) *AsyncWriter {
	return &AsyncWriter{Writer: NewWriter(w)}
}

type AsyncWriter struct {
	*Writer

	amu        sync.Mutex
	schedule   chan<- struct{}
	unschedule chan<- struct{}
}

// WriteLineAsync writes a single protocol message. The message won't be written immediately
// and will be batched with other similar messages.
func (w *AsyncWriter) WriteLineAsync(data []byte) error {
	w.amu.Lock()
	defer w.amu.Unlock()

	if w.schedule != nil {
		// batch already started
		err := w.Writer.WriteLine(data)
		// wake flush routine if it's blocked
		select {
		case w.schedule <- struct{}{}:
		default:
		}
		return err
	}

	err := w.Writer.StartBatch()
	if err != nil {
		return err
	}
	err = w.Writer.WriteLine(data)
	if err != nil {
		_ = w.Writer.EndBatch(false)
		return err
	}

	re := make(chan struct{}, 1)
	un := make(chan struct{})
	w.schedule = re
	w.unschedule = un

	go func() {
		const delay = time.Millisecond * 15
		timer := time.NewTimer(delay)
		defer timer.Stop()
		for {
			select {
			case <-un:
				return
			case <-timer.C:
				w.amu.Lock()
				// we may have missed the unschedule event
				select {
				case <-un:
				default:
					_ = w.Writer.EndBatch(false)
					w.schedule = nil
					w.unschedule = nil
				}
				w.amu.Unlock()
				return
			case <-re:
				timer.Reset(delay)
			}
		}
	}()
	return nil
}

// Flush waits for all async writes to complete and forces the flush of internal buffers.
func (w *AsyncWriter) Flush() error {
	w.amu.Lock()
	defer w.amu.Unlock()
	if w.schedule == nil {
		return w.Writer.Flush()
	}
	// routine will now exit, we don't have to wait for it
	close(w.unschedule)
	w.schedule = nil
	w.unschedule = nil
	if err := w.Writer.EndBatch(true); err != nil {
		return err
	}
	return w.Writer.Flush()
}
