package lineproto

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const delim = '|'

func TestConnRW(t *testing.T) {
	for _, readLoop := range []bool{false, true} {
		for _, concurrent := range []bool{false, true} {
			name := "linear"
			if readLoop {
				name = "read loop"
			}
			if concurrent {
				name += " conc"
			}
			t.Run(name, func(t *testing.T) {
				c1, c2 := net.Pipe()
				defer c1.Close()
				defer c2.Close()

				cw := NewConn(c1, delim)
				cr := NewConn(c2, delim)

				var (
					wg sync.WaitGroup

					wmu sync.Mutex
					exp []string

					got []string
				)
				addExp := func(s string) {
					wmu.Lock()
					exp = append(exp, s)
					wmu.Unlock()
				}
				writeLoop := func(w *BatchWriter, i int) {
					for j := 0; j < 5; j++ {
						line := fmt.Sprintf("line %d-%d|", i, j)
						err := w.WriteLine([]byte(line))
						require.NoError(t, err)
						addExp(line)
					}
					err := w.Close()
					require.NoError(t, err)
				}

				wg.Add(1)
				go func(c *Conn) {
					defer wg.Done()
					defer c.Close()

					if concurrent {
						var wwg sync.WaitGroup
						for i := 0; i < 5; i++ {
							i := i
							wwg.Add(1)
							go func() {
								defer wwg.Done()

								w, err := c.BeginWrite()
								require.NoError(t, err)
								writeLoop(w, i)
							}()
						}
						wwg.Wait()
					} else {
						for i := 0; i < 5; i++ {
							w, err := c.BeginWrite()
							require.NoError(t, err)
							writeLoop(w, i)
						}
					}

					line := "end|"
					err := c.CloseWith([]byte(line))
					require.NoError(t, err)
					addExp(line)
				}(cw)

				wg.Add(1)
				go func(c *Conn) {
					defer wg.Done()
					defer c.Close()

					if readLoop {
						c.OnLineR(func(line []byte) (bool, error) {
							got = append(got, string(line))
							return true, nil
						})
						err := c.ReadLoop()
						if err == io.EOF || err == io.ErrClosedPipe {
							return
						}
						require.NoError(t, err)
					} else {
						for {
							line, err := c.ReadLine()
							if err == io.EOF || err == io.ErrClosedPipe {
								return
							}
							require.NoError(t, err)
							got = append(got, string(line))
						}
					}
				}(cr)

				done := make(chan struct{})
				go func() {
					defer close(done)
					wg.Wait()
				}()
				select {
				case <-time.After(10 * time.Second):
					t.Fatal("timeout")
				case <-done:
					require.Equal(t, exp, got)
				}
			})
		}
	}
}

func TestAsyncKeepAlive(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	cw := NewConn(c1, delim)
	cr := c2

	line := []byte{delim}
	err := cw.KeepAlive(line, time.Millisecond)
	require.NoError(t, err)
	var (
		got  bytes.Buffer
		errc = make(chan error, 1)
	)
	go func() {
		defer close(errc)
		_, err := io.Copy(&got, cr)
		if err != nil && err != io.ErrClosedPipe {
			errc <- err
		}
	}()
	time.Sleep(15 * time.Millisecond)
	err = cw.Close()
	require.NoError(t, err)
	err = cr.Close()
	require.NoError(t, err)
	err = <-errc
	require.NoError(t, err)
	exp := bytes.Repeat(line, 15)
	require.True(t, got.Len() >= 10 && got.Len() <= 16)
	require.Equal(t, exp[:got.Len()], got.Bytes())
}

func TestAsyncWriter(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	cw := NewConn(c1, delim)
	cr := c2

	var (
		wg   sync.WaitGroup
		errc = make(chan error, 1)
	)
	for i := 0; i < 10; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				err := func() error {
					w, err := cw.BeginWriteAsync()
					if err != nil {
						return err
					}
					err = w.WriteLine([]byte(fmt.Sprintf("line %d-%d"+string(delim), i, j)))
					if err != nil {
						return err
					}
					return w.Close()
				}()
				if err != nil {
					select {
					case errc <- err:
					default:
					}
					return
				}
			}
		}()
	}
	var (
		got   bytes.Buffer
		errcr = make(chan error, 1)
	)
	go func() {
		defer close(errcr)
		_, err := io.Copy(&got, cr)
		if err != nil && err != io.ErrClosedPipe {
			errcr <- err
		}
	}()
	wg.Wait()
	err := cw.Close()
	require.NoError(t, err)
	select {
	case err := <-errc:
		require.NoError(t, err)
	default:
	}
	err = <-errcr
	require.NoError(t, err)
	lines := strings.Split(got.String(), string(delim))
	sort.Strings(lines)
	var exp []string
	exp = append(exp, "")
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			exp = append(exp, fmt.Sprintf("line %d-%d", i, j))
		}
	}
	sort.Strings(exp)
	require.Equal(t, exp, lines)
}

func TestSyncWriter(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	cw := NewConn(c1, delim)
	cr := c2

	w, err := cw.BeginWrite()
	require.NoError(t, err)

	// this write must lock because we are not reading anything
	errc := make(chan error, 1)
	go func() {
		defer close(errc)
		if err := w.WriteLine([]byte("test" + string(delim))); err != nil {
			errc <- err
			return
		}
		if err := w.Flush(); err != nil {
			errc <- err
			return
		}
	}()
	select {
	case <-errc:
		t.Fatal("must lock")
	case <-time.After(time.Second / 2):
	}
	// unblock
	buf := make([]byte, 10)
	n, err := cr.Read(buf)
	require.NoError(t, err)
	require.Equal(t, "test"+string(delim), string(buf[:n]))
	select {
	case err := <-errc:
		require.NoError(t, err)
	case <-time.After(time.Second / 2):
		t.Fatal("timeout")
	}
	errc = make(chan error, 1)
	// this will block, but should be released by Drop
	go func() {
		defer close(errc)
		if err := w.WriteLine([]byte("test2" + string(delim))); err != nil {
			errc <- err
			return
		}
		if err := w.Flush(); err != nil {
			errc <- err
			return
		}
	}()
	select {
	case <-errc:
		t.Fatal("must lock")
	case <-time.After(time.Second / 4):
	}
	// unblock
	cw.Drop()
	select {
	case err := <-errc:
		require.Equal(t, errWriterClosed, err)
	case <-time.After(time.Second / 2):
		t.Fatal("timeout")
	}
	// nothing was written
	n, err = cr.Read(buf)
	require.Equal(t, io.EOF, err)
	require.Equal(t, 0, n)
}

func TestWriterAndCloseWith(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	cw := NewConn(c1, delim)
	cr := c2

	var (
		got  bytes.Buffer
		errc = make(chan error, 1)
	)
	go func() {
		defer close(errc)
		_, err := io.Copy(&got, cr)
		if err != nil && err != io.ErrClosedPipe {
			errc <- err
		}
	}()

	w, err := cw.BeginWrite()
	require.NoError(t, err)

	err = w.WriteLine([]byte("test1" + string(delim)))
	require.NoError(t, err)
	err = w.Flush()
	require.NoError(t, err)
	// don't close it, try closing the connection instead

	err = cw.CloseWith([]byte("end" + string(delim)))
	require.NoError(t, err)

	err = <-errc
	require.NoError(t, err)
	require.Equal(t, "test1|end|", got.String())
}
