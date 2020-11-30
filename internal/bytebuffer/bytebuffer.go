package bytebuffer

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"
)

const maxCap = 1 << 12

type pool struct {
	cnt   int64
	inner sync.Pool
}

func (bp *pool) get(size int) *ByteBuff {
	atomic.AddInt64(&bp.cnt, 1)
	if exist, _ := bp.inner.Get().(*ByteBuff); exist != nil {
		if size > 0 {
			((*bytes.Buffer)(exist)).Grow(size)
		}
		return exist
	}
	bb := new(bytes.Buffer)
	if size > 0 {
		bb.Grow(size)
	}
	return (*ByteBuff)(bb)
}

func (bp *pool) put(bb *ByteBuff) {
	if bb != nil && ((*bytes.Buffer)(bb)).Cap() <= maxCap {
		bb.Reset()
		bp.inner.Put(bb)
	}
	atomic.AddInt64(&bp.cnt, -1)
}

func (bp *pool) size() int64 {
	return atomic.LoadInt64(&bp.cnt)
}

var global pool

// ByteBuff provides byte buffer, which can be used for minimizing.
type ByteBuff bytes.Buffer

// CountBorrowed returns borrowed ByteBuff amount.
func CountBorrowed() int64 {
	return global.size()
}

// BorrowByteBuff borrows a ByteBuff from pool.
func BorrowByteBuff(n int) *ByteBuff {
	return global.get(n)
}

// ReturnByteBuff returns a ByteBuff to pool.
func ReturnByteBuff(b *ByteBuff) {
	global.put(b)
}

// Reset resets ByteBuff.
func (b *ByteBuff) Reset() {
	b.bb().Reset()
}

// Len returns size of ByteBuff.
func (b *ByteBuff) Len() (n int) {
	return b.bb().Len()
}

// WriteTo writes bytes to writer.
func (b *ByteBuff) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(b.bb().Bytes())
	return int64(n), err
}

// Write writes bytes to current ByteBuff.
func (b *ByteBuff) Write(bs []byte) (int, error) {
	return b.bb().Write(bs)
}

// WriteByte writes a byte to current ByteBuff.
func (b *ByteBuff) WriteByte(c byte) error {
	return b.bb().WriteByte(c)
}

// WriteString writes a string to current ByteBuff.
func (b *ByteBuff) WriteString(s string) (err error) {
	_, err = b.bb().WriteString(s)
	return
}

// Bytes returns all bytes in ByteBuff.
func (b *ByteBuff) Bytes() []byte {
	return b.bb().Bytes()
}

func (b *ByteBuff) bb() *bytes.Buffer {
	return (*bytes.Buffer)(b)
}
