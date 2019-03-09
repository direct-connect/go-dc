package lineproto

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
)

const (
	readBuf = 4096
	maxLine = readBuf * 8
)

var (
	errValueIsTooLong = errors.New("value is too long")
)

type ErrProtocolViolation struct {
	Err error
}

func (e *ErrProtocolViolation) Error() string {
	return fmt.Sprintf("protocol error: %v", e.Err)
}

func NewReader(r io.Reader, delim byte) *Reader {
	return &Reader{
		r: r, buf: make([]byte, 0, readBuf),
		delim:   delim,
		maxLine: maxLine,
	}
}

// Reader is safe for concurrent use.
type Reader struct {
	r     io.Reader
	delim byte

	maxLine int

	mu   sync.Mutex
	err  error
	buf  []byte
	i    int
	mbuf bytes.Buffer // TODO

	// OnLine is called each time a raw protocol line is read from the connection.
	// The buffer will contain a delimiter and is in the connection encoding.
	// The function may return (false, nil) to ignore the message.
	OnLine func(line []byte) (bool, error)
}

// SetMaxLine sets a maximal length of the protocol messages in bytes, including the delimiter.
func (r *Reader) SetMaxLine(n int) {
	r.maxLine = n
}

func (r *Reader) peek() ([]byte, error) {
	if r.i < len(r.buf) {
		return r.buf[r.i:], nil
	}
	r.i = 0
	r.buf = r.buf[:cap(r.buf)]
	n, err := r.r.Read(r.buf)
	r.buf = r.buf[:n]
	return r.buf, err
}

func (r *Reader) discard(n int) {
	if n < 0 {
		r.i += len(r.buf)
	} else {
		r.i += n
	}
}

// readUntil reads a byte slice until the delimiter, up to max bytes.
// It returns a newly allocated slice with a delimiter and reads bytes and the delimiter
// from the reader.
func (r *Reader) readUntil(delim string, max int) ([]byte, error) {
	r.mbuf.Reset()
	for {
		b, err := r.peek()
		if err != nil {
			return nil, err
		}
		i := bytes.Index(b, []byte(delim))
		if i >= 0 {
			r.mbuf.Write(b[:i+len(delim)])
			r.discard(i + len(delim))
			return r.mbuf.Bytes(), nil
		}
		if r.mbuf.Len()+len(b) > max {
			return nil, errValueIsTooLong
		}
		r.mbuf.Write(b)
		r.discard(-1)
	}
}

// ReadLine reads a single raw message until the delimiter. The returned buffer contains
// a delimiter and is in the connection encoding. The buffer is only valid until the next
// call to Read or ReadLine.
func (r *Reader) ReadLine() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err; err != nil {
		return nil, err
	}
	for {
		data, err := r.readUntil(string(r.delim), r.maxLine)
		if err == errValueIsTooLong {
			r.err = &ErrProtocolViolation{
				Err: fmt.Errorf("cannot read message: %v", err),
			}
			return nil, r.err
		} else if err != nil {
			r.err = err
			return nil, err
		}
		if r.OnLine != nil {
			if ok, err := r.OnLine(data); err != nil {
				r.err = err
				return nil, err
			} else if !ok {
				continue // drop
			}
		}
		return data, nil
	}
}
