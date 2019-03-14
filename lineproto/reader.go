package lineproto

import (
	"bufio"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"sync"
)

const (
	readBuf = 2048 // TCP MTU is ~1500
	maxLine = readBuf * 8
)

var errorBufferExhausted = errors.New("message buffer exhausted")

type ErrProtocolViolation struct {
	Err error
}

func (e *ErrProtocolViolation) Error() string {
	return fmt.Sprintf("protocol error: %v", e.Err)
}

type fullReader interface {
	io.ByteReader
	io.Reader
}

// zlibSwitchableReader is a zlib reader that can be switched on at any time.
// It requires a io.ByteReader, otherwise zlib adds a bufio reader that
// messes up the switching phase.
type zlibSwitchableReader struct {
	in           fullReader
	zlibReader   io.ReadCloser
	activeReader io.Reader
}

func newZlibSwitchableReader(in fullReader) *zlibSwitchableReader {
	return &zlibSwitchableReader{
		in:           in,
		activeReader: in,
	}
}

func (c *zlibSwitchableReader) Read(buf []byte) (int, error) {
	for {
		n, err := c.activeReader.Read(buf)

		// zlib EOF: disable and read again
		if n == 0 && err == io.EOF && c.activeReader == c.zlibReader {
			c.zlibReader.Close()
			c.activeReader = c.in
			continue
		}
		return n, err
	}
}

func (c *zlibSwitchableReader) ActivateCompression() error {
	if c.activeReader == c.zlibReader {
		return fmt.Errorf("zlib already activated")
	}

	var err error
	if c.zlibReader == nil {
		c.zlibReader, err = zlib.NewReader(c.in)
	} else {
		err = c.zlibReader.(zlib.Resetter).Reset(c.in, nil)
	}
	if err != nil {
		return err
	}
	c.activeReader = c.zlibReader
	return nil
}

// Reader is a line reader that supports the zlib on/off switching procedure
// required by NMDC hub-to-client and client-to-client connections.
type Reader struct {
	in    *zlibSwitchableReader
	delim byte
	mutex sync.Mutex

	// Safe can be set to disable internal mutex.
	Safe bool

	// OnLine is called each time a raw protocol line is read from the connection.
	// The buffer will contain a delimiter and is in the connection encoding.
	// The function may return (false, nil) to ignore the message.
	OnLine func(line []byte) (bool, error)
}

// NewReader allocates a Reader.
func NewReader(in io.Reader, delim byte) *Reader {
	// first reader is a buffer that provides the io.ByteReader interface
	l1 := bufio.NewReaderSize(in, readBuf)

	// second reader is zlibSwitchableReader
	l2 := newZlibSwitchableReader(l1)

	// third reader is the line reader
	return &Reader{
		in:    l2,
		delim: delim,
	}
}

// ReadLine reads a single raw message until the delimiter. The returned buffer contains
// a delimiter and is in the connection encoding. The buffer is only valid until the next
// call to Read or ReadLine.
func (r *Reader) ReadLine() ([]byte, error) {
	if r.Safe == false {
		r.mutex.Lock()
		defer r.mutex.Unlock()
	}

	buf := make([]byte, maxLine)
	offset := 0
	for {
		if offset >= len(buf) {
			return nil, errorBufferExhausted
		}

		// read one character at a time
		read, err := r.in.Read(buf[offset : offset+1])
		if read == 0 {
			return nil, err
		}
		offset++

		if buf[offset-1] != r.delim {
			continue
		}

		if r.OnLine != nil {
			// OnLine() error
			if ok, err := r.OnLine(buf[:offset]); err != nil {
				return nil, err

				// OnLine() commanded to drop buffer
			} else if !ok {
				offset = 0
				continue
			}
		}

		return buf[:offset], nil
	}
}

// ActivateCompression activates zlib deflating.
func (r *Reader) ActivateCompression() error {
	return r.in.ActivateCompression()
}
