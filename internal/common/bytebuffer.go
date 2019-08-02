package common

import (
	"io"
)

type ByteBuff struct {
	// B is a byte buffer to use in append-like workloads.
	// See example code for details.
	B []byte
}

// Len returns size of ByteBuff.
func (p *ByteBuff) Len() (n int) {
	if p != nil {
		n = len(p.B)
	}
	return
}

// WriteTo write bytes to writer.
func (p *ByteBuff) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(p.B)
	return int64(n), err
}

// Writer write bytes to current ByteBuff.
func (p *ByteBuff) Write(bs []byte) (int, error) {
	p.B = append(p.B, bs...)
	return len(bs), nil
}

// WriteUint24 encode and write Uint24 to current ByteBuff.
func (p *ByteBuff) WriteUint24(n int) (err error) {
	v := NewUint24(n)
	_, err = p.Write(v[:])
	return
}

// WriteByte write a byte to current ByteBuff.
func (p *ByteBuff) WriteByte(b byte) error {
	p.B = append(p.B, b)
	return nil
}

// Reset clean all bytes.
func (p *ByteBuff) Reset() {
	p.B = p.B[:0]
}

// Bytes returns all bytes in ByteBuff.
func (p *ByteBuff) Bytes() []byte {
	return p.B
}

// New borrows a ByteBuff from pool.
func New() (bb *ByteBuff) {
	return &ByteBuff{}
}
