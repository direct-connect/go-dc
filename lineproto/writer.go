package lineproto

import (
	"bufio"
	"compress/zlib"
	"errors"
	"io"
)

var (
	errWriterClosed = errors.New("writer is closed")
)

func NewWriter(w io.Writer) *Writer {
	return NewWriterSize(w, 0)
}

func NewWriterSize(w io.Writer, buf int) *Writer {
	return &Writer{
		w: w, cur: w, bw: bufio.NewWriterSize(w, buf),
	}
}

// Writer is not safe for concurrent use.
type Writer struct {
	// Timeout is a callback to setup timeout on each write.
	Timeout func(enable bool) error

	// onLine is called each time a raw protocol message is written.
	// The function may return (false, nil) to skip writing the message.
	onLine []func(line []byte) (bool, error)

	w       io.Writer
	cur     io.Writer     // active writer underlying the bw
	bw      *bufio.Writer // buffered writer on top of cur
	err     error
	zlibOn  bool
	zlibW   *zlib.Writer
	zlibLvl int
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

// Err returns the last encountered error.
func (w *Writer) Err() error {
	return w.err
}

// Close flushes the writer and frees its resources. It won't close the underlying writer.
func (w *Writer) Close() error {
	last := w.bw.Flush()
	if w.err == nil {
		w.err = errWriterClosed
	}
	if w.zlibOn {
		if err := w.zlibW.Close(); err != nil {
			last = err
		}
		w.zlibOn = false
		w.zlibW = nil
	}
	w.bw = nil
	w.cur = nil
	w.w = nil
	w.onLine = nil
	return last
}

// EnableZlib activates zlib deflating.
func (w *Writer) EnableZlib() error {
	return w.EnableZlibLevel(zlib.DefaultCompression)
}

// EnableZlib activates zlib deflating with a given compression level.
func (w *Writer) EnableZlibLevel(lvl int) error {
	if w.err != nil {
		return w.err
	} else if w.zlibOn {
		return errZlibAlreadyActive
	}
	if err := w.Flush(); err != nil {
		return err
	}
	w.zlibOn = true
	if w.zlibW == nil || w.zlibLvl != lvl {
		z, err := zlib.NewWriterLevel(w.w, lvl)
		if err != nil {
			return err
		}
		w.zlibW = z
		w.zlibLvl = lvl
	} else {
		w.zlibW.Reset(w.w)
	}
	w.cur = w.zlibW
	w.bw.Reset(w.cur)
	return nil
}

// DisableZlib deactivates zlib deflating.
func (w *Writer) DisableZlib() error {
	if w.err != nil {
		return w.err
	} else if !w.zlibOn {
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
	w.cur = w.w
	w.bw.Reset(w.cur)
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

// Write the data to the connection. It will flush any remaining buffer and will bypass
// buffering for this call.
func (w *Writer) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if w.bw.Buffered() != 0 {
		if err := w.Flush(); err != nil {
			return 0, err
		}
	}
	return w.cur.Write(p)
}

// WriteLine writes a single protocol message.
func (w *Writer) WriteLine(data []byte) error {
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
	if w.Timeout != nil {
		err := w.Timeout(true)
		if err != nil {
			w.setError(err)
			return err
		}
		defer w.Timeout(false)
	}
	_, err := w.bw.Write(data)
	if err != nil {
		w.setError(err)
		return err
	}
	return nil
}
