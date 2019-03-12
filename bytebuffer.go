package rsocket

import (
	"github.com/valyala/bytebufferpool"
	"io"
	"sync/atomic"
	"time"
)

const trackLeakSeconds = 10

var (
	pool    bytebufferpool.Pool
	borrows int64
)

func init() {
	// for test
	if trackLeakSeconds < 1 {
		return
	}
	tk := time.NewTicker(trackLeakSeconds * time.Second)
	go func() {
		var stop bool
		for {
			if stop {
				break
			}
			select {
			case _, ok := <-tk.C:
				if !ok {
					stop = true
				}
				logger.Debugf("#### bytebuff borrows: %d\n", borrows)
			}
		}
	}()
}

type rByteBuffer bytebufferpool.ByteBuffer

func (p *rByteBuffer) Len() int {
	return p.bb().Len()
}

func (p *rByteBuffer) WriteTo(w io.Writer) (n int64, err error) {
	return p.bb().WriteTo(w)
}

func (p *rByteBuffer) Write(bs []byte) (n int, err error) {
	return p.bb().Write(bs)
}

func (p *rByteBuffer) WriteUint24(n int) (err error) {
	foo := newUint24(n)
	for i := 0; i < 3; i++ {
		err = p.WriteByte(foo[i])
		if err != nil {
			break
		}
	}
	return
}

func (p *rByteBuffer) WriteByte(b byte) error {
	return p.bb().WriteByte(b)
}

func (p *rByteBuffer) Reset() {
	p.bb().Reset()
}

func (p *rByteBuffer) Bytes() []byte {
	if p.bb() == nil {
		return nil
	}
	return p.bb().B
}

func (p *rByteBuffer) bb() *bytebufferpool.ByteBuffer {
	return (*bytebufferpool.ByteBuffer)(p)
}

func borrowByteBuffer() *rByteBuffer {
	if trackLeakSeconds > 0 {
		defer func() {
			atomic.AddInt64(&borrows, 1)
		}()
	}
	return (*rByteBuffer)(pool.Get())
}

func returnByteBuffer(b *rByteBuffer) {
	if trackLeakSeconds > 0 {
		defer func() {
			atomic.AddInt64(&borrows, -1)
		}()
	}
	pool.Put((*bytebufferpool.ByteBuffer)(b))
}
