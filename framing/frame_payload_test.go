package framing

import (
	"bytes"
	"github.com/rsocket/rsocket-go/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFramePayload_Basic(t *testing.T) {
	metadata := []byte("hello")
	data := []byte("world")
	f := NewFramePayload(123, data, metadata)

	assert.Equal(t, metadata, f.Metadata(), "metadata failed")
	assert.Equal(t, data, f.Data(), "data failed")

	bf := &bytes.Buffer{}
	_, _ = f.WriteTo(bf)
	bs := bf.Bytes()

	bb := common.BorrowByteBuffer()
	_, _ = bb.Write(bs[HeaderLen:])
	f2 := &FramePayload{
		&BaseFrame{
			header: ParseFrameHeader(bs[:HeaderLen]),
			body:   bb,
		},
	}
	assert.Equal(t, metadata, f2.Metadata(), "metadata failed 2")
	assert.Equal(t, data, f2.Data(), "data failed 2")
	assert.Equal(t, f.header[:], f2.header[:])
}
