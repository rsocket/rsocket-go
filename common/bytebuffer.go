package common

import (
	"github.com/valyala/bytebufferpool"
	"io"
)

var bPool bytebufferpool.Pool

// ByteBuff provides byte buffer, which can be used for minimizing.
type ByteBuff bytebufferpool.ByteBuffer

// Len returns size of ByteBuff.
func (p *ByteBuff) Len() int {
	return p.bb().Len()
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
	foo := NewUint24(n)
	for i := 0; i < 3; i++ {
		err = p.WriteByte(foo[i])
		if err != nil {
			break
		}
	}
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
func BorrowByteBuffer() *ByteBuff {
	return (*ByteBuff)(bPool.Get())
}

// ReturnByteBuffer returns a ByteBuff.
func ReturnByteBuffer(b *ByteBuff) {
	bPool.Put((*bytebufferpool.ByteBuffer)(b))
}
