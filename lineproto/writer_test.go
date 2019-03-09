package lineproto

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

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
