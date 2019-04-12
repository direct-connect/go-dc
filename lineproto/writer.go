package lineproto

import (
	"bufio"
	"compress/zlib"
	"io"
)

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: w, bw: bufio.NewWriter(w),
	}
}

// Writer is not safe for concurrent use.
type Writer struct {
	// Timeout is a callback to setup timeout on each write.
	Timeout func(enable bool) error

	// onLine is called each time a raw protocol message is written.
	// The function may return (false, nil) to skip writing the message.
	onLine []func(line []byte) (bool, error)

	w      io.Writer
	bw     *bufio.Writer
	err    error
	zlibOn bool
	zlibW  *zlib.Writer
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
}

func (w *Writer) Err() error {
	return w.err
}

// EnableZlib activates zlib deflating.
func (w *Writer) EnableZlib() error {
	if w.zlibOn {
		return errZlibAlreadyActive
	}
	if err := w.Flush(); err != nil {
		return err
	}
	w.zlibOn = true
	if w.zlibW == nil {
		w.zlibW = zlib.NewWriter(w.w)
	} else {
		w.zlibW.Reset(w.w)
	}
	w.bw.Reset(w.zlibW)
	return nil
}

// DisableZlib deactivates zlib deflating.
func (w *Writer) DisableZlib() error {
	if !w.zlibOn {
		return errZlibNotActive
	}
	err := w.bw.Flush()
	if err != nil {
		w.setError(err)
		return err
	}
	err = w.zlibW.Close()
	if err != nil {
		w.setError(err)
		return err
	}
	w.zlibOn = false
	w.bw.Reset(w.w)
	return nil
}

// Flush forces all buffered writes to be flushed.
func (w *Writer) Flush() error {
	if w.err != nil {
		return w.err
	}
	if w.Timeout != nil {
		err := w.Timeout(true)
		if err != nil {
			w.setError(err)
			return err
		}
		defer w.Timeout(false)
	}
	err := w.bw.Flush()
	if err != nil {
		w.setError(err)
		return err
	}
	if !w.zlibOn {
		return nil
	}
	err = w.zlibW.Flush()
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
	for _, fnc := range w.onLine {
		if ok, err := fnc(data); err != nil {
			w.setError(err)
			return err
		} else if !ok {
			return nil
		}
	}
	if w.Timeout != nil {
		err := w.Timeout(true)
		if err != nil {
			w.setError(err)
			return err
		}
		defer w.Timeout(false)
	}
	// buffer not empty - write to it
	_, err := w.bw.Write(data)
	if err != nil {
		w.setError(err)
		return err
	}
	return nil
}
