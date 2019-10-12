package common

import (
	"bytes"
	"io"
)

// ByteBuff provides byte buffer, which can be used for minimizing.
type ByteBuff bytes.Buffer

func (p *ByteBuff) pp() *bytes.Buffer {
	return (*bytes.Buffer)(p)
}

// Len returns size of ByteBuff.
func (p *ByteBuff) Len() (n int) {
	return p.pp().Len()
}

// WriteTo write bytes to writer.
func (p *ByteBuff) WriteTo(w io.Writer) (int64, error) {
	return p.pp().WriteTo(w)
}

// Writer write bytes to current ByteBuff.
func (p *ByteBuff) Write(bs []byte) (int, error) {
	return p.pp().Write(bs)
}

// WriteUint24 encode and write Uint24 to current ByteBuff.
func (p *ByteBuff) WriteUint24(n int) (err error) {
	v := NewUint24(n)
	_, err = p.Write(v[:])
	return
}

// WriteByte write a byte to current ByteBuff.
func (p *ByteBuff) WriteByte(b byte) error {
	return p.pp().WriteByte(b)
}

// Bytes returns all bytes in ByteBuff.
func (p *ByteBuff) Bytes() []byte {
	return p.pp().Bytes()
}

// NewByteBuff creates a new ByteBuff.
func NewByteBuff() *ByteBuff {
	return (*ByteBuff)(&bytes.Buffer{})
}
