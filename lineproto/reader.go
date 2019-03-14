package lineproto

import (
	"bufio"
	"compress/flate"
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
var errorZlibAlreadyActive = errors.New("zlib already activated")

type ErrProtocolViolation struct {
	Err error
}

func (e *ErrProtocolViolation) Error() string {
	return fmt.Sprintf("protocol error: %v", e.Err)
}

// zlibSwitchableReader is a zlib reader that can be switched on at any time.
// the flate.Reader requirement ensures that zlib read only the necessary bytes,
// leaving the remainings in the previous buffer.
type zlibSwitchableReader struct {
	r                  flate.Reader
	zlibReader         io.ReadCloser
	zlibActive         bool
	zlibBufferedReader io.ByteReader
}

func newZlibSwitchableReader(r flate.Reader) *zlibSwitchableReader {
	return &zlibSwitchableReader{
		r: r,
	}
}

func (c *zlibSwitchableReader) ActivateZlib() error {
	if c.zlibActive == true {
		return errorZlibAlreadyActive
	}
	c.zlibActive = true

	// allocate a new reader
	if c.zlibReader == nil {
		var err error
		c.zlibReader, err = zlib.NewReader(c.r)
		if err != nil {
			return err
		}
		c.zlibBufferedReader = bufio.NewReaderSize(c.zlibReader, readBuf)
		return nil
	}

	// reuse previous reader
	return c.zlibReader.(zlib.Resetter).Reset(c.r, nil)
}

func (c *zlibSwitchableReader) ReadByte() (byte, error) {
	if c.zlibActive == false {
		return c.r.ReadByte()
	}

	res, err := c.zlibBufferedReader.ReadByte()

	// zlib EOF: disable zlib and read again
	if err == io.EOF {
		c.zlibActive = false
		return c.r.ReadByte()
	}

	return res, err
}

// Reader is a line reader that supports the zlib on/off switching procedure
// required by hub-to-client and client-to-client connections.
type Reader struct {
	r      *zlibSwitchableReader
	delim  byte
	mutex  sync.Mutex
	buffer []byte

	// Safe can be set to disable internal mutex.
	Safe bool

	// OnLine is called each time a raw protocol line is read from the connection.
	// The buffer will contain a delimiter and is in the connection encoding.
	// The function may return (false, nil) to ignore the message.
	OnLine func(line []byte) (bool, error)
}

// NewReader allocates a Reader.
func NewReader(r io.Reader, delim byte) *Reader {
	// first reader is bufio.Reader
	l1 := bufio.NewReaderSize(r, readBuf)

	// second reader is zlibSwitchableReader
	l2 := newZlibSwitchableReader(l1)

	// third reader is the line reader
	return &Reader{
		r:      l2,
		delim:  delim,
		buffer: make([]byte, maxLine),
	}
}

// ReadLine reads a single raw message until the delimiter. The returned buffer contains
// a delimiter and is in the connection encoding. The buffer is only valid until the next
// call to Read or ReadLine.
func (r *Reader) ReadLine() ([]byte, error) {
	if !r.Safe {
		r.mutex.Lock()
		defer r.mutex.Unlock()
	}

	offset := 0
	for {
		if offset >= len(r.buffer) {
			return nil, errorBufferExhausted
		}

		// transfer one byte at a time
		var err error
		r.buffer[offset], err = r.r.ReadByte()
		if err != nil {
			return nil, err
		}
		offset++

		if r.buffer[offset-1] != r.delim {
			continue
		}

		if r.OnLine != nil {
			// OnLine() error
			if ok, err := r.OnLine(r.buffer[:offset]); err != nil {
				return nil, err

				// OnLine() commanded to drop buffer
			} else if !ok {
				offset = 0
				continue
			}
		}

		return r.buffer[:offset], nil
	}
}

// ActivateZlib activates zlib deflating.
func (r *Reader) ActivateZlib() error {
	return r.r.ActivateZlib()
}
