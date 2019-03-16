package framing

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHeader_All(t *testing.T) {
	h1 := NewFrameHeader(134, FrameTypePayload, FlagMetadata|FlagComplete|FlagNext)
	h := ParseFrameHeader(h1[:])
	assert.Equal(t, h1.StreamID(), h.StreamID())
	assert.Equal(t, h1.Type(), h.Type())
	assert.Equal(t, h1.Flag(), h.Flag())

	fmt.Println("streamID:", h.StreamID())
	fmt.Println("type:", h.Type())
	fmt.Println("flag:", h)
}
