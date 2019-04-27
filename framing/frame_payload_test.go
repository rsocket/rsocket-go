package framing

import (
	"bytes"
	"testing"

	"github.com/rsocket/rsocket-go/common"
	"github.com/stretchr/testify/assert"
)

func TestFramePayload_Basic(t *testing.T) {
	metadata := []byte("hello")
	data := []byte("world")
	f := NewFramePayload(123, data, metadata)

	metadata1, _ := f.Metadata()
	assert.Equal(t, metadata, metadata1, "metadata failed")
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

	metadata2, _ := f2.Metadata()
	assert.Equal(t, metadata, metadata2, "metadata failed 2")
	assert.Equal(t, data, f2.Data(), "data failed 2")
	assert.Equal(t, f.header[:], f2.header[:])
}
