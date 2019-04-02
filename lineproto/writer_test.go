package lineproto

import (
	"bytes"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
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
		0xcf, 0xcd, 0x4d, 0xcc, 0x4b, 0x51, 0x28, 0x49, 0x2d, 0x2e, 0xa9, 0x1, 0x0, 0x0,
		0x0, 0xff, 0xff, 0x1, 0x0, 0x0, 0xff, 0xff, 0x45, 0x1e, 0x7, 0x42}...)
	exp = append(exp, []byte("$Uncompressed|$ZOn2|")...)
	exp = append(exp, []byte{0x78, 0x9c, 0x52, 0xf1, 0x2f, 0xc9, 0x48, 0x2d, 0x72, 0xce,
		0xcf, 0xcd, 0x4d, 0xcc, 0x4b, 0x51, 0x28, 0x49, 0x2d, 0x2e, 0xa9, 0x1, 0x0, 0x0,
		0x0, 0xff, 0xff, 0x52, 0x71, 0xce, 0xcf, 0x2d, 0x28, 0x4a, 0x2d, 0x2e, 0x4e, 0x4d,
		0x31, 0xaa, 0x1, 0x0, 0x0, 0x0, 0xff, 0xff, 0x0, 0x0, 0x0, 0xff, 0xff}...)
	require.Equal(t, exp, buf.Bytes())

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
	}
}

func BenchmarkWriteBatch(b *testing.B) {
	d := &Discard{dt: time.Millisecond}
	data := make([]byte, 1024)
	w := NewWriter(d)

	b.ResetTimer()
	err := w.StartBatch()
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		err := w.WriteLine(data)
		if err != nil {
			b.Fatal(err)
		}
	}
	err = w.EndBatch(true)
	if err != nil {
		b.Fatal(err)
	}
}

func BenchmarkWriteBatchConcurrent(b *testing.B) {
	d := &Discard{dt: time.Millisecond}
	data := make([]byte, 1024)
	w := NewWriter(d)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_ = w.StartBatch()
			_ = w.WriteLine(data)
			_ = w.EndBatch(false)
		}()
	}
	b.ResetTimer()
	close(start)
	wg.Wait()
}

func BenchmarkWriteAsync(b *testing.B) {
	d := &Discard{dt: time.Millisecond}
	data := make([]byte, 1024)
	w := NewAsyncWriter(d)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := w.WriteLineAsync(data)
		if err != nil {
			b.Fatal(err)
		}
	}
	err := w.Flush()
	if err != nil {
		b.Fatal(err)
	}
}

func BenchmarkWriteAsyncConcurrent(b *testing.B) {
	d := &Discard{dt: time.Millisecond}
	data := make([]byte, 1024)
	w := NewAsyncWriter(d)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_ = w.WriteLineAsync(data)
		}()
	}
	b.ResetTimer()
	close(start)
	wg.Wait()
	err := w.Flush()
	if err != nil {
		b.Fatal(err)
	}
}
