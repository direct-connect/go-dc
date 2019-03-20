package lineproto

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"sync"
)

const (
	readBuf = 2048 // TCP MTU is ~1500
	maxLine = readBuf * 16
)

var (
	errBufferExhausted   = errors.New("message is too long")
	errZlibAlreadyActive = errors.New("zlib already activated")
)

type ErrProtocolViolation struct {
	Err error
}

func (e *ErrProtocolViolation) Error() string {
	return fmt.Sprintf("protocol error: %v", e.Err)
}

var (
	_ io.Reader    = (*bufReader)(nil)
	_ flate.Reader = (*bufReader)(nil)
)

func newBufReader(r io.Reader) *bufReader {
	return &bufReader{
		r: r, buf: make([]byte, 0, readBuf),
	}
}

// bufReader is a copy of bufio.Reader that allows to expose the underlying buffer.
type bufReader struct {
	r   io.Reader
	buf []byte // pre-allocated, never grows
	off int    // offset into buf
}

func (r *bufReader) Reset(rd io.Reader) {
	r.r = rd
	r.off = 0
	r.buf = r.buf[:0]
}

func (r *bufReader) peek() error {
	n, err := r.r.Read(r.buf[:cap(r.buf)])
	if n == 0 && err != nil {
		return err
	}
	// suppress error if we get any data
	// it will be returned on the next call
	r.buf = r.buf[:n]
	r.off = 0
	return nil
}

func (r *bufReader) ReadByte() (byte, error) {
	if r.off >= len(r.buf) {
		if err := r.peek(); err != nil {
			return 0, err
		}
	}
	b := r.buf[r.off]
	r.off++
	return b, nil
}

func (r *bufReader) Read(p []byte) (int, error) {
	if r.off < len(r.buf) {
		n := copy(p, r.buf[r.off:])
		r.off += n
		return n, nil
	}
	// bypass the buffer
	return r.r.Read(p)
}

func (r *bufReader) Scan(delim byte) ([]byte, bool, error) {
	if r.off >= len(r.buf) {
		if err := r.peek(); err != nil {
			return nil, false, err
		}
	}
	buf := r.buf[r.off:]
	i := bytes.IndexByte(buf, delim)
	if i < 0 {
		r.off += len(buf)
		// need more bytes
		return buf, true, nil
	}
	// found in the buffer
	buf = buf[:i+1]
	r.off += len(buf)
	return buf, false, nil
}

// Reader is a line reader that supports the zlib on/off switching procedure
// required by hub-to-client and client-to-client connections.
type Reader struct {
	delim byte

	mu         sync.Mutex
	cur        *bufReader // current reader; set either to original or compressed
	original   *bufReader // original reader with buffer
	zlibOn     bool
	zlib       io.ReadCloser // resettable zlib reader stored for reuse
	compressed *bufReader    // compressed reader; stored for reuse
	line       []byte        // buffered line; up to maxLine bytes

	// Safe can be set to disable internal mutex.
	Safe bool

	// OnLine is called each time a raw protocol line is read from the connection.
	// The buffer will contain a delimiter and is in the connection encoding.
	// The function may return (false, nil) to ignore the message.
	OnLine func(line []byte) (bool, error)
}

// NewReader allocates a Reader.
func NewReader(r io.Reader, delim byte) *Reader {
	br := newBufReader(r)
	return &Reader{
		delim:    delim,
		original: br,
		cur:      br,
		line:     make([]byte, readBuf),
	}
}

// ReadLine reads a single raw message until the delimiter. The returned buffer contains
// a delimiter and is in the connection encoding. The buffer is only valid until the next
// call to Read or ReadLine.
func (r *Reader) ReadLine() ([]byte, error) {
	if !r.Safe {
		r.mu.Lock()
		defer r.mu.Unlock()
	}
	r.line = r.line[:0]

	for {
		if len(r.line) >= maxLine {
			return nil, errBufferExhausted
		}
		pref, more, err := r.cur.Scan(r.delim)
		if err == io.EOF && r.zlibOn {
			// if compression was enabled, we need to switch back to original reader
			r.cur = r.original
			r.zlibOn = false
			continue
		}
		r.line = append(r.line, pref...)
		if err != nil {
			return r.line, err
		} else if more {
			continue
		}

		line := r.line
		if r.OnLine != nil {
			if ok, err := r.OnLine(line); err != nil {
				return nil, err
			} else if !ok {
				// hook commands to drop the message
				r.line = r.line[:0]
				continue
			}
		}

		return line, nil
	}
}

// Read reads a byte slice, inflates it if zlib is active, and puts the
// result into buf.
func (r *Reader) Read(buf []byte) (int, error) {
	n, err := r.cur.Read(buf)
	if err == io.EOF && r.zlibOn {
		// if compression was enabled, we need to switch back to original reader
		r.cur = r.original
		r.zlibOn = false

		// if some data was read, return it without errors.
		if n > 0 {
			return n, nil
		}

		// no data was read. Read again.
		return r.cur.Read(buf)
	}
	return n, err
}

// ActivateZlib activates zlib inflating.
func (r *Reader) ActivateZlib() error {
	if r.zlibOn {
		return errZlibAlreadyActive
	}
	r.zlibOn = true

	if r.zlib != nil {
		err := r.zlib.(zlib.Resetter).Reset(r.original, nil)
		if err != nil {
			return err
		}
	} else {
		rc, err := zlib.NewReader(r.original)
		if err != nil {
			return err
		}
		r.zlib = rc
	}

	if r.compressed != nil {
		r.compressed.Reset(r.zlib)
	} else {
		r.compressed = newBufReader(r.zlib)
	}
	r.cur = r.compressed
	return nil
}
