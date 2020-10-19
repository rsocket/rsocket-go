package common

import (
	"bytes"
	"io"
	"sync"

	"go.uber.org/atomic"
)

var _byteBuffPool = sync.Pool{
	New: func() interface{} {
		return new(ByteBuff)
	},
}

var _borrowed = atomic.NewInt64(0)

// ByteBuff provides byte buffer, which can be used for minimizing.
type ByteBuff bytes.Buffer

// CountBorrowed returns borrowed ByteBuff amount.
func CountBorrowed() int64 {
	return _borrowed.Load()
}

// BorrowByteBuff borrows a ByteBuff from pool.
func BorrowByteBuff() *ByteBuff {
	_borrowed.Inc()
	b := _byteBuffPool.Get().(*ByteBuff)
	return b
}

// ReturnByteBuff returns a ByteBuff to pool.
func ReturnByteBuff(b *ByteBuff) {
	_borrowed.Dec()
	b.Reset()
	_byteBuffPool.Put(b)
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
