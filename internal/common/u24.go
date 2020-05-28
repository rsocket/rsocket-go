package common

import (
	"errors"
	"fmt"
	"io"
)

// MaxUint24 is the max value of Uint24.
const MaxUint24 = 16777215

var (
	errExceedMaxUint24 = fmt.Errorf("uint24 exceed max value: %d", MaxUint24)
	errNegativeNumber  = errors.New("negative number is illegal")
)

// IsExceedMaximumUint24Error returns true if exceed maximum Uint24. (16777215)
func IsExceedMaximumUint24Error(err error) bool {
	return err == errExceedMaxUint24
}

// IsNegativeUint24Error returns true if number is negative.
func IsNegativeUint24Error(err error) bool {
	return err == errNegativeNumber
}

// Uint24 is 3 bytes unsigned integer.
type Uint24 [3]byte

// Bytes returns bytes encoded.
func (p Uint24) Bytes() []byte {
	return p[:]
}

// WriteTo encode and write bytes to a writer.
func (p Uint24) WriteTo(w io.Writer) (int64, error) {
	wrote, err := w.Write(p[:])
	return int64(wrote), err
}

// AsInt converts to int.
func (p Uint24) AsInt() int {
	return int(p[0])<<16 + int(p[1])<<8 + int(p[2])
}

// MustNewUint24 returns a new uint24.
func MustNewUint24(n int) Uint24 {
	v, err := NewUint24(n)
	if err != nil {
		panic(err)
	}
	return v
}

// NewUint24 returns a new uint24.
func NewUint24(v int) (n Uint24, err error) {
	if v < 0 {
		err = errNegativeNumber
		return
	}
	if v > MaxUint24 {
		err = errExceedMaxUint24
	}
	n[0] = byte(v >> 16)
	n[1] = byte(v >> 8)
	n[2] = byte(v)
	return
}

// NewUint24Bytes returns a new uint24 from bytes.
func NewUint24Bytes(bs []byte) Uint24 {
	_ = bs[2]
	return [3]byte{bs[0], bs[1], bs[2]}
}
