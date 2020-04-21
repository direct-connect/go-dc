package lineproto

import (
	"bytes"
	"compress/zlib"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type connConfig struct {
	WriteBuffer int
}

// ConnOption is an option for the connection.
type ConnOption interface {
	apply(c *connConfig)
}

type connOptionFunc func(c *connConfig)

func (f connOptionFunc) apply(c *connConfig) {
	f(c)
}

// WithWriteBuffer sets the write buffer size for the connection.
func WithWriteBuffer(buf int) ConnOption {
	return connOptionFunc(func(c *connConfig) {
		c.WriteBuffer = buf
	})
}

// NewConn creates a new line protocol connection with a specified delimiter.
func NewConn(conn net.Conn, delim byte, opts ...ConnOption) *Conn {
	var conf connConfig
	for _, opt := range opts {
		opt.apply(&conf)
	}
	return &Conn{
		c: conn,
		w: NewWriterSize(conn, conf.WriteBuffer),
		r: NewReader(conn, delim),
	}
}

type Conn struct {
	c        net.Conn
	closed   int32 // atomic
	readLoop int32 // atomic

	wmu sync.Mutex
	w   *Writer

	worker struct {
		sync.Mutex
		err       error
		stop      chan struct{}      // signal to stop the worker
		done      chan struct{}      // signals when the worker exits
		trigger   chan struct{}      // signals that there is a new buffer ready
		keepAlive chan keepAliveConf // changes the keep alive config for the worker
		writers   int                // number of open async writers
		buf       *bytes.Buffer
	}

	rmu sync.Mutex
	r   *Reader
}

func (c *Conn) isClosed() bool {
	return atomic.LoadInt32(&c.closed) != 0
}

func (c *Conn) setClosed() bool {
	return atomic.CompareAndSwapInt32(&c.closed, 0, 1)
}

func (c *Conn) isInReadLoop() bool {
	return atomic.LoadInt32(&c.readLoop) != 0
}

func (c *Conn) setReadLoop() bool {
	return atomic.CompareAndSwapInt32(&c.readLoop, 0, 1)
}

// LocalAddr returns local address of the underlying connection.
func (c *Conn) LocalAddr() net.Addr {
	return c.c.LocalAddr()
}

// RemoteAddr returns remote address of the underlying connection.
func (c *Conn) RemoteAddr() net.Addr {
	return c.c.RemoteAddr()
}

// OnLineR sets reader's line hook. See Reader.OnLine for details.
func (c *Conn) OnLineR(fnc func(line []byte) (bool, error)) {
	if c.isInReadLoop() {
		// precaution
		panic("multiple read loops detected")
	}
	c.rmu.Lock()
	c.r.OnLine(fnc)
	c.rmu.Unlock()
}

// OnLineW sets writer's line hook. See Writer.OnLine for details.
func (c *Conn) OnLineW(fnc func(line []byte) (bool, error)) {
	c.wmu.Lock()
	c.w.OnLine(fnc)
	c.wmu.Unlock()
}

func (c *Conn) SetReadDeadline(deadline time.Time) error {
	if c.isClosed() {
		return errReaderClosed
	}
	c.rmu.Lock()
	defer c.rmu.Unlock()
	if c.isClosed() {
		return errReaderClosed
	}
	return c.c.SetReadDeadline(deadline)
}

// ReadLine reads a protocol line and optionally sets the deadline for this read.
func (c *Conn) ReadLine() ([]byte, error) {
	if c.isClosed() {
		return nil, errReaderClosed
	}
	c.rmu.Lock()
	defer c.rmu.Unlock()
	if c.isClosed() {
		return nil, errReaderClosed
	}
	return c.r.ReadLine()
}

// ReadLoop starts a read loop which blocks any other reads.
// Instead, it calls all the callbacks set with OnLineR.
// It is not safe to call Drop from the callback. Return an error instead.
func (c *Conn) ReadLoop() error {
	if !c.setReadLoop() {
		// precaution
		panic("multiple read loops detected")
	}
	c.rmu.Lock()
	defer c.rmu.Unlock()
	for !c.isClosed() {
		if _, err := c.r.ReadLine(); err != nil {
			if c.isClosed() {
				return nil
			}
			return err
		}
	}
	return nil
}

// EnableZlibR enables Zlib compression for the reader.
func (c *Conn) EnableZlibR() error {
	if c.isClosed() {
		return errReaderClosed
	}
	c.rmu.Lock()
	defer c.rmu.Unlock()
	if c.isClosed() {
		return errReaderClosed
	}
	return c.r.EnableZlib()
}

// EnableZlibW sends the specified line and enables Zlib compression for the writer.
func (c *Conn) EnableZlibW(line []byte, lvl int) error {
	if lvl <= 0 {
		lvl = zlib.DefaultCompression
	}
	if c.isClosed() {
		return errWriterClosed
	}
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if c.isClosed() {
		return errWriterClosed
	}
	if len(line) != 0 {
		if err := c.w.WriteLine(line); err != nil {
			return err
		}
	}
	// flushes
	return c.w.EnableZlibLevel(lvl)
}

// BeginWrite starts a batched write to the connection.
// Users must not keep the writer open for too long - it may interfere with other writers and Close.
func (c *Conn) BeginWrite() (*BatchWriter, error) {
	if c.isClosed() {
		return nil, errWriterClosed
	}
	// try locking it
	c.wmu.Lock()
	err := c.w.Err()
	c.wmu.Unlock()
	if err != nil {
		return nil, err
	}
	return &BatchWriter{
		c: c,
		w: c.w, locked: false,
	}, nil
}

func (c *Conn) closeW() error {
	c.wmu.Lock()
	err := c.w.Close()
	c.wmu.Unlock()
	return err
}

func (c *Conn) closeR() error {
	c.rmu.Lock()
	err := c.r.Close()
	c.rmu.Unlock()
	return err
}

// Drop immediately terminates the connection. It won't send any buffers, and all reads and writes will fail.
func (c *Conn) Drop() {
	if !c.setClosed() {
		return
	}
	c.stopWorker(false)
	_ = c.c.Close()
	_ = c.closeR()
	_ = c.closeW()
}

// dropW is the same as Drop, but assumes the write mutex was already acquired.
func (c *Conn) dropW() {
	if !c.setClosed() {
		return
	}
	c.stopWorker(false)
	_ = c.c.Close()
	_ = c.closeR()
	_ = c.w.Close()
}

// Close will wait for all writes to finish before closing the connection.
func (c *Conn) Close() error {
	return c.CloseWith()
}

// CloseWith is similar to Close, but it will also send one or more protocol lines just before closing the connection.
func (c *Conn) CloseWith(lines ...[]byte) error {
	c.stopWorker(true)
	// wait for all writers
	c.wmu.Lock()
	defer c.wmu.Unlock()

	if c.isClosed() {
		// connection dropped
		return errWriterClosed
	}
	defer c.dropW()

	if err := c.c.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return err
	}
	for _, line := range lines {
		if err := c.w.WriteLine(line); err != nil {
			return err
		}
	}
	return c.w.Flush()
}

var _ LineWriter = (*BatchWriter)(nil)

type BatchWriter struct {
	closed   bool
	c        *Conn
	locked   bool
	w        *Writer
	deadline time.Time
}

func (w *BatchWriter) isClosed() bool {
	if w.c.isClosed() {
		w.close()
		return true
	}
	return w.closed
}

func (w *BatchWriter) ensureLocked() error {
	if w.isClosed() {
		return errWriterClosed
	}
	if w.locked {
		return nil
	}
	w.c.wmu.Lock()
	if w.isClosed() {
		w.c.wmu.Unlock()
		return errWriterClosed
	}
	w.locked = true
	return nil
}

// WriteLine writes a protocol line to the connection.
// It may be partially buffered and flushed periodically.
// Call Flush to ensure that the data was sent.
func (w *BatchWriter) WriteLine(data []byte) error {
	if err := w.ensureLocked(); err != nil {
		return err
	}
	if err := w.w.WriteLine(data); err != nil {
		if w.isClosed() {
			return errWriterClosed
		}
		w.Drop()
		return err
	}
	// still locked, released by Flush, Drop or Close
	return nil
}

func (w *BatchWriter) SetWriteDeadline(deadline time.Time) error {
	if err := w.ensureLocked(); err != nil {
		return err
	}
	if err := w.c.c.SetWriteDeadline(deadline); err != nil {
		w.Drop()
		return err
	}
	w.deadline = deadline
	// still locked, released by Flush, Drop or Close
	return nil
}

func (w *BatchWriter) SetWriteTimeout(timeout time.Duration) error {
	return w.SetWriteDeadline(time.Now().Add(timeout))
}

// Flush any pending writes.
func (w *BatchWriter) Flush() error {
	if err := w.ensureLocked(); err != nil {
		return err
	}
	if err := w.w.Flush(); err != nil {
		if w.isClosed() {
			return errWriterClosed
		}
		w.Drop()
		return err
	}
	w.c.wmu.Unlock()
	w.locked = false
	return nil
}

func (w *BatchWriter) close() {
	if !w.closed {
		w.closed = true
		if !w.deadline.IsZero() {
			_ = w.c.c.SetWriteDeadline(time.Time{})
		}
		if w.locked {
			w.locked = false
			w.c.wmu.Unlock()
		}
	}
}

// Drop the writer without flushing the rest of the data.
// The connection will be left in undefined state since some data may have already been flushed.
// Thus this function will also drop the underlying connection.
func (w *BatchWriter) Drop() {
	if !w.closed {
		w.c.dropW()
		w.close()
	}
}

// Close the writer and flush all the buffered data. It will close the writer, but not the connection.
func (w *BatchWriter) Close() error {
	if w.isClosed() {
		return errWriterClosed
	}
	err := w.Flush()
	w.close() // only releases the writer
	return err
}

type keepAliveConf struct {
	Line     []byte
	Interval time.Duration
}

func (c *Conn) KeepAlive(line []byte, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Minute / 2
	}
	c.worker.Lock()
	if err := c.worker.err; err != nil {
		c.worker.Unlock()
		return err
	}
	if c.worker.done == nil {
		if err := c.startWorker(); err != nil {
			c.worker.err = err
			c.worker.Unlock()
			return err
		}
	}
	done, keepAlive := c.worker.done, c.worker.keepAlive
	c.worker.Unlock()
	select {
	case <-done:
		return errWriterClosed
	case keepAlive <- keepAliveConf{Line: line, Interval: interval}:
		return nil
	}
}

func (c *Conn) writer(w *BatchWriter, stop <-chan struct{}) {
	var werr error
	defer func() {
		c.worker.Lock()
		c.worker.err = werr
		done := c.worker.done
		c.worker.done = nil
		close(done)
		c.worker.Unlock()
		w.close() // only releases the writer
	}()

	var keepAlive struct {
		timer *time.Timer
		keepAliveConf
	}
	keepAlive.timer = time.NewTimer(time.Second)
	keepAlive.timer.Stop()

	const timeout = 15 * time.Second
	var (
		prevBuf  *bytes.Buffer
		deadline time.Time
	)
	for {
		select {
		case <-stop:
			return
		case <-keepAlive.timer.C:
			if len(keepAlive.Line) == 0 {
				continue
			}
			deadline = time.Now().Add(timeout)
			werr = w.SetWriteDeadline(deadline)
			if werr != nil {
				return
			}
			werr = w.WriteLine(keepAlive.Line)
			if werr != nil {
				return
			}
			werr = w.Flush()
			if werr != nil {
				return
			}
		case conf := <-c.worker.keepAlive:
			keepAlive.keepAliveConf = conf
		case <-c.worker.trigger:
			c.worker.Lock()
			buf := c.worker.buf
			if buf == prevBuf {
				c.worker.Unlock()
				panic("buffer swap failure")
			}
			c.worker.buf = prevBuf
			c.worker.Unlock()
			if buf == nil || buf.Len() == 0 {
				prevBuf = buf
				continue
			}

			start := time.Now()
			if start.After(deadline) || deadline.Sub(start) < (timeout*3)/4 {
				// set only if an old deadline is stale
				deadline = start.Add(timeout)
				werr = w.SetWriteDeadline(deadline)
				if werr != nil {
					return
				}
			}

			werr = w.WriteLine(buf.Bytes())
			if werr != nil {
				return
			}
			werr = w.Flush()
			if werr != nil {
				return
			}
			buf.Reset()
			prevBuf = buf
		}
		keepAlive.timer.Reset(keepAlive.Interval)
	}
}

// startWorker locks sync writes and spawns a worker goroutine which will serve all async writers.
// This function must be called while holding the worker mutex.
func (c *Conn) startWorker() error {
	// TODO: this should wait for old async writers to exit
	w, err := c.BeginWrite()
	if err != nil {
		return err
	}
	c.worker.stop = make(chan struct{})
	c.worker.done = make(chan struct{})
	c.worker.trigger = make(chan struct{}, 1)
	c.worker.keepAlive = make(chan keepAliveConf)
	go c.writer(w, c.worker.stop)
	return nil
}

func (c *Conn) stopWorker(wait bool) {
	c.worker.Lock()
	done := c.worker.done
	stop := c.worker.stop
	if done != nil && stop != nil {
		close(c.worker.stop)
		c.worker.stop = nil
	}
	c.worker.Unlock()
	if wait && done != nil {
		<-done
	}
}

func (c *Conn) BeginWriteAsync() (*AsyncWriter, error) {
	c.worker.Lock()
	if err := c.worker.err; err != nil {
		c.worker.Unlock()
		return nil, err
	}
	if c.worker.done == nil {
		if err := c.startWorker(); err != nil {
			c.worker.err = err
			c.worker.Unlock()
			return nil, err
		}
	}
	c.worker.writers++
	done, trigger := c.worker.done, c.worker.trigger
	c.worker.Unlock()
	return &AsyncWriter{
		c:       c,
		done:    done,
		trigger: trigger,
	}, nil
}

type AsyncWriter struct {
	c       *Conn
	closed  bool
	done    <-chan struct{}
	trigger chan<- struct{}
}

func (w *AsyncWriter) WriteLine(line []byte) error {
	select {
	case <-w.done:
		return errWriterClosed
	default:
	}
	w.c.worker.Lock()
	defer w.c.worker.Unlock()
	if w.c.worker.err != nil {
		err := w.close()
		return err
	}
	if w.c.worker.buf == nil {
		w.c.worker.buf = bytes.NewBuffer(nil)
	}
	w.c.worker.buf.Write(line)
	return nil
}

func (w *AsyncWriter) close() error {
	if !w.closed {
		w.c.worker.writers--
		w.closed = true
	}
	return w.c.worker.err
}

func (w *AsyncWriter) Flush() error {
	select {
	case w.trigger <- struct{}{}:
	default:
	}
	return nil
}

func (w *AsyncWriter) Close() error {
	w.c.worker.Lock()
	err := w.close()
	w.c.worker.Unlock()
	if err == nil {
		_ = w.Flush()
	}
	return err
}
