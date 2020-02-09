package lineproto

import (
	"bytes"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterZlib(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	w := NewWriter(buf)

	var lines []string

	write := func(s string) {
		err := w.WriteLine([]byte(s))
		require.NoError(t, err)
		lines = append(lines, s)
	}

	write("$ZOn|")

	err := w.EnableZlib()
	require.NoError(t, err)

	write("$OtherCommand test|")

	// full close of zlib
	err = w.DisableZlib()
	require.NoError(t, err)

	write("$Uncompressed|")

	write("$ZOn2|")

	err = w.EnableZlib()
	require.NoError(t, err)

	write("$OtherCommand test|")
	write("$Compressed2|")

	// partial flush, zlib still open
	err = w.Flush()
	require.NoError(t, err)

	var exp []byte
	exp = append(exp, []byte("$ZOn|")...)
	exp = append(exp, []byte{0x78, 0x9c, 0x52, 0xf1, 0x2f, 0xc9, 0x48, 0x2d, 0x72, 0xce,
		0xcf, 0xcd, 0x4d, 0xcc, 0x4b, 0x51, 0x28, 0x49, 0x2d, 0x2e, 0xa9, 0x1, 0x4, 0x0,
		0x0, 0xff, 0xff, 0x45, 0x1e, 0x7, 0x42}...)
	exp = append(exp, []byte("$Uncompressed|$ZOn2|")...)
	exp = append(exp, []byte{0x78, 0x9c, 0x52, 0xf1, 0x2f, 0xc9, 0x48, 0x2d, 0x72, 0xce,
		0xcf, 0xcd, 0x4d, 0xcc, 0x4b, 0x51, 0x28, 0x49, 0x2d, 0x2e, 0xa9, 0x51, 0x71, 0xce,
		0xcf, 0x2d, 0x28, 0x4a, 0x2d, 0x2e, 0x4e, 0x4d, 0x31, 0xaa, 0x1, 0x0, 0x0, 0x0,
		0xff, 0xff}...)
	assert.Equal(t, exp, buf.Bytes())

	r := NewReader(buf, '|')
	for _, exp := range lines {
		l, err := r.ReadLine()
		require.NoError(t, err)
		require.Equal(t, exp, string(l))
		if strings.HasPrefix(exp, "$ZOn") {
			err = r.EnableZlib()
			require.NoError(t, err)
		}
	}

	// add more data to make sure zlib is still active
	write("$Compressed3|")
	err = w.Flush()
	require.NoError(t, err)

	l, err := r.ReadLine()
	require.NoError(t, err)
	require.Equal(t, "$Compressed3|", string(l))
	require.True(t, r.zlibOn)
}

func TestWriterZlibClose(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	w := NewWriter(buf)

	var lines []string

	write := func(s string) {
		err := w.WriteLine([]byte(s))
		require.NoError(t, err)
		lines = append(lines, s)
	}

	err := w.EnableZlib()
	require.NoError(t, err)

	write("$CommandOne test1|")
	err = w.Flush()
	require.NoError(t, err)
	write("$CommandTwo test2|")

	err = w.Close()
	require.NoError(t, err)

	r := NewReader(buf, '|')
	err = r.EnableZlib()
	require.NoError(t, err)

	var got []string
	for {
		line, err := r.ReadLine()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		got = append(got, string(line))
	}
	require.Equal(t, lines, got)
}

func TestWriterTimeout(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	w := NewWriter(c1)
	w.Timeout = func(enable bool) error {
		if enable {
			return c1.SetWriteDeadline(time.Now().Add(time.Second / 4))
		}
		return c1.SetWriteDeadline(time.Time{})
	}

	err := w.WriteLine([]byte("Test|"))
	require.NoError(t, err) // buffer
	err = w.Flush()
	require.NotNil(t, err)
	if e, ok := err.(interface {
		Timeout() bool
	}); !ok || !e.Timeout() {
		require.FailNowf(t, "expected timeout error", "got: %T, %v", err, err)
	}
}

type Discard struct {
	n  uint32
	dt time.Duration
}

func (d *Discard) Write(b []byte) (int, error) {
	if d.dt != 0 {
		time.Sleep(d.dt)
	}
	atomic.AddUint32(&d.n, uint32(len(b)))
	return len(b), nil
}

func BenchmarkWriteLine(b *testing.B) {
	d := &Discard{dt: time.Millisecond}
	data := make([]byte, 1024)
	w := NewWriter(d)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := w.WriteLine(data)
		if err != nil {
			b.Fatal(err)
		}
		err = w.Flush()
		if err != nil {
			b.Fatal(err)
		}
	}
}
