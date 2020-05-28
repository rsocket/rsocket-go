package common

import (
	"bytes"
	"io"
)

// ByteBuff provides byte buffer, which can be used for minimizing.
type ByteBuff bytes.Buffer

func (b *ByteBuff) pp() *bytes.Buffer {
	return (*bytes.Buffer)(b)
}

// Len returns size of ByteBuff.
func (b *ByteBuff) Len() (n int) {
	return b.pp().Len()
}

// WriteTo write bytes to writer.
func (b *ByteBuff) WriteTo(w io.Writer) (int64, error) {
	return b.pp().WriteTo(w)
}

// Writer write bytes to current ByteBuff.
func (b *ByteBuff) Write(bs []byte) (int, error) {
	return b.pp().Write(bs)
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
	return b.pp().WriteByte(c)
}

// WriteString writes a string to current ByteBuff.
func (b *ByteBuff) WriteString(s string) (err error) {
	_, err = b.pp().Write([]byte(s))
	return
}

// Bytes returns all bytes in ByteBuff.
func (b *ByteBuff) Bytes() []byte {
	return b.pp().Bytes()
}

// NewByteBuff creates a new ByteBuff.
func NewByteBuff() *ByteBuff {
	return (*ByteBuff)(&bytes.Buffer{})
}
