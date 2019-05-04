package common

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	"github.com/rsocket/rsocket-go/internal/logger"
	"github.com/valyala/bytebufferpool"
)

var (
	borrowed int32
	bPool    bytebufferpool.Pool
)

// ByteBuff provides byte buffer, which can be used for minimizing.
type ByteBuff bytebufferpool.ByteBuffer

// Len returns size of ByteBuff.
func (p *ByteBuff) Len() (n int) {
	if p != nil {
		n = p.bb().Len()
	}
	return
}

// WriteTo write bytes to writer.
func (p *ByteBuff) WriteTo(w io.Writer) (n int64, err error) {
	return p.bb().WriteTo(w)
}

// Writer write bytes to current ByteBuff.
func (p *ByteBuff) Write(bs []byte) (n int, err error) {
	return p.bb().Write(bs)
}

// WriteUint24 encode and write Uint24 to current ByteBuff.
func (p *ByteBuff) WriteUint24(n int) (err error) {
	v := NewUint24(n)
	_, err = p.Write(v[:])
	return
}

// WriteByte write a byte to current ByteBuff.
func (p *ByteBuff) WriteByte(b byte) error {
	return p.bb().WriteByte(b)
}

// Reset clean all bytes.
func (p *ByteBuff) Reset() {
	p.bb().Reset()
}

// Bytes returns all bytes in ByteBuff.
func (p *ByteBuff) Bytes() []byte {
	if p.bb() == nil {
		return nil
	}
	return p.bb().B
}

func (p *ByteBuff) bb() *bytebufferpool.ByteBuffer {
	return (*bytebufferpool.ByteBuffer)(p)
}

// BorrowByteBuffer borrows a ByteBuff from pool.
func BorrowByteBuffer() (bb *ByteBuff) {
	bb = (*ByteBuff)(bPool.Get())
	atomic.AddInt32(&borrowed, 1)
	return
}

// ReturnByteBuffer returns a ByteBuff.
func ReturnByteBuffer(b *ByteBuff) {
	bPool.Put((*bytebufferpool.ByteBuffer)(b))
	atomic.AddInt32(&borrowed, -1)
}

// CountByteBuffer returns amount of ByteBuff borrowed.
func CountByteBuffer() int {
	return int(atomic.LoadInt32(&borrowed))
}

func TraceByteBuffLeak(ctx context.Context, duration time.Duration) error {
	tk := time.NewTicker(duration)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tk.C:
			logger.Infof("=====> count bytebuffers: %d\n", CountByteBuffer())
		}
	}
}
