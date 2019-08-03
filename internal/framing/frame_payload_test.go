package framing

import (
	"bytes"
	"testing"

	"github.com/rsocket/rsocket-go/internal/common"
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

	bb := common.New()
	_, _ = bb.Write(bs[HeaderLen:])
	f2 := &FramePayload{
		NewBaseFrame(ParseFrameHeader(bs[:HeaderLen]), bb),
	}

	metadata2, _ := f2.Metadata()
	assert.Equal(t, metadata, metadata2, "metadata failed 2")
	assert.Equal(t, data, f2.Data(), "data failed 2")
	assert.Equal(t, f2.header[:], f2.header[:])
}
