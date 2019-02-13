package rsocket

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFramePayload_Basic(t *testing.T) {
	metadata := []byte("hello")
	data := []byte("world")
	f := createPayloadFrame(123, data, metadata)

	assert.Equal(t, metadata, f.Metadata(), "metadata failed")
	assert.Equal(t, data, f.Data(), "data failed")

	bf := &bytes.Buffer{}
	_, _ = f.WriteTo(bf)
	bs := bf.Bytes()

	bb := borrowByteBuffer()
	_, _ = bb.Write(bs[headerLen:])
	f2 := &framePayload{
		&baseFrame{
			header: parseHeaderBytes(bs[:headerLen]),
			body:   bb,
		},
	}
	assert.Equal(t, metadata, f2.Metadata(), "metadata failed 2")
	assert.Equal(t, data, f2.Data(), "data failed 2")
	assert.Equal(t, f.header[:], f2.header[:])
}
