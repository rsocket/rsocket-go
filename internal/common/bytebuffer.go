package common

import (
	"io"

	"github.com/valyala/bytebufferpool"
	"go.uber.org/atomic"
)

var _borrowed = atomic.NewInt64(0)

// ByteBuff provides byte buffer, which can be used for minimizing.
type ByteBuff bytebufferpool.ByteBuffer

// CountBorrowed returns borrowed ByteBuff amount.
func CountBorrowed() int64 {
	return _borrowed.Load()
}

// BorrowByteBuff borrows a ByteBuff from pool.
func BorrowByteBuff() *ByteBuff {
	_borrowed.Inc()
	return (*ByteBuff)(bytebufferpool.Get())
}

// ReturnByteBuff returns a ByteBuff to pool.
func ReturnByteBuff(b *ByteBuff) {
	_borrowed.Dec()
	bytebufferpool.Put((*bytebufferpool.ByteBuffer)(b))
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
	return b.bb().WriteTo(w)
}

// Write writes bytes to current ByteBuff.
func (b *ByteBuff) Write(bs []byte) (int, error) {
	return b.bb().Write(bs)
}

// WriteUint24 encode and write Uint24 to current ByteBuff.
func (b *ByteBuff) WriteUint24(n int) (err error) {
	if n > MaxUint24 {
		return errExceedMaxUint24
	}
	v := MustNewUint24(n)
	_, err = b.Write(v[:])
	return
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

func (b *ByteBuff) bb() *bytebufferpool.ByteBuffer {
	return (*bytebufferpool.ByteBuffer)(b)
}
