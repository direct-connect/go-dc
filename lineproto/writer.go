package lineproto

import (
	"bufio"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: w, bw: bufio.NewWriter(w),
	}
}

// Writer is safe for concurrent use.
type Writer struct {
	// onLine is called each time a raw protocol message is written.
	// The function may return (false, nil) to skip writing the message.
	onLine []func(line []byte) (bool, error)

	errNolock atomic.Value // synced with err

	mu  sync.Mutex
	w   io.Writer
	bw  *bufio.Writer
	err error
	lvl int
}

// OnLine registers a hook that is called each time a raw protocol message is written.
// The function may return (false, nil) to skip writing the message.
//
// This method is not concurrent-safe.
func (w *Writer) OnLine(fnc func(line []byte) (bool, error)) {
	w.onLine = append(w.onLine, fnc)
}

func (w *Writer) setError(err error) {
	w.err = err
	w.errNolock.Store(err)
}

func (w *Writer) Err() error {
	v, _ := w.errNolock.Load().(error)
	return v
}

func (w *Writer) flush() error {
	err := w.bw.Flush()
	if err != nil {
		w.setError(err)
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
	for _, fnc := range w.onLine {
		if ok, err := fnc(data); err != nil {
			w.setError(err)
			return err
		} else if !ok {
			return nil
		}
	}
	if w.lvl > 0 {
		// someone will flush for us
		_, err := w.bw.Write(data)
		if err != nil {
			w.setError(err)
		}
		return err
	}
	if w.bw.Size() != 0 {
		// buffer not empty - write through it
		_, err := w.bw.Write(data)
		if err != nil {
			w.setError(err)
			return err
		}
		return w.flush()
	}
	// empty buffer - write directly
	_, err := w.w.Write(data)
	if err != nil {
		w.setError(err)
	}
	return err
}

// WriteLine writes a single protocol message.
func (w *Writer) WriteLine(data []byte) error {
	if err := w.Err(); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writeLine(data)
}

func (w *Writer) startOrWrite(data []byte) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.err; err != nil {
		return false, err
	}
	_, err := w.bw.Write(data)
	if err != nil {
		w.setError(err)
		return false, err
	}
	if w.lvl > 0 {
		// someone will flush for us
		return false, nil
	}
	// batch
	w.lvl++
	return true, nil
}

// StartBatch starts a batch of messages. Caller should call EndBatch to flush the buffer.
func (w *Writer) StartBatch() error {
	if err := w.Err(); err != nil {
		return err
	}
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
	schedule uint32 // atomic

	amu        sync.Mutex
	unschedule chan<- struct{}
}

// WriteLineAsync writes a single protocol message. The message won't be written immediately
// and will be batched with other similar messages.
func (w *AsyncWriter) WriteLineAsync(data []byte) error {
	if err := w.Err(); err != nil {
		return err
	}

	w.amu.Lock()
	defer w.amu.Unlock()

	if w.unschedule != nil {
		// routine already started
		err := w.Writer.WriteLine(data)
		atomic.AddUint32(&w.schedule, 1)
		return err
	}

	batch, err := w.Writer.startOrWrite(data)
	if err != nil {
		return err
	} else if !batch {
		return nil
	}

	un := make(chan struct{})
	atomic.StoreUint32(&w.schedule, 0)
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
				// check if more writes were scheduled during the sleep
				if v := atomic.LoadUint32(&w.schedule); v != 0 {
					// sleep more, let others to fill and flush the buffer
					atomic.CompareAndSwapUint32(&w.schedule, v, 0)
					timer.Reset(delay)
					continue
				}
				w.amu.Lock()
				if v := atomic.LoadUint32(&w.schedule); v != 0 {
					w.amu.Unlock()
					atomic.CompareAndSwapUint32(&w.schedule, v, 0)
					timer.Reset(delay)
					continue
				}
				// we may have missed the unschedule event
				select {
				case <-un:
				default:
					_ = w.Writer.EndBatch(false)
					atomic.StoreUint32(&w.schedule, 0)
					w.unschedule = nil
				}
				w.amu.Unlock()
				return
			}
		}
	}()
	return nil
}

// Flush waits for all async writes to complete and forces the flush of internal buffers.
func (w *AsyncWriter) Flush() error {
	if err := w.Err(); err != nil {
		return err
	}

	w.amu.Lock()
	defer w.amu.Unlock()
	if w.unschedule == nil {
		return w.Writer.Flush()
	}
	// routine will now exit, we don't have to wait for it
	close(w.unschedule)
	atomic.StoreUint32(&w.schedule, 0)
	w.unschedule = nil
	if err := w.Writer.EndBatch(true); err != nil {
		return err
	}
	return w.Writer.Flush()
}
