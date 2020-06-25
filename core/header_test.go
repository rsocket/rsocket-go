package core_test

import (
	"bytes"
	"math"
	"testing"

	. "github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestHeader_All(t *testing.T) {
	id := uint32(common.RandIntn(math.MaxUint32))
	h1 := NewFrameHeader(id, FrameTypePayload, FlagMetadata|FlagComplete|FlagNext)
	assert.NotEmpty(t, h1.String(), "header string is blank")
	assert.True(t, h1.Resumable())
	h2 := ParseFrameHeader(h1[:])
	assert.Equal(t, h1[:], h2.Bytes())
	assert.Equal(t, h1.StreamID(), h2.StreamID())
	assert.Equal(t, h1.Type(), h2.Type())
	assert.Equal(t, h1.Flag(), h2.Flag())
	assert.Equal(t, FrameTypePayload, h1.Type())
	assert.Equal(t, FlagMetadata|FlagComplete|FlagNext, h1.Flag())
	bf := &bytes.Buffer{}
	n, err := h2.WriteTo(bf)
	assert.NoError(t, err)
	assert.Equal(t, int64(FrameHeaderLen), n)
}
