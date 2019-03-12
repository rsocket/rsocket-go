package rsocket

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHeader_All(t *testing.T) {
	h1 := createHeader(134, tPayload, flagMetadata|flagComplete|flagNext)
	h := parseHeaderBytes(h1[:])
	assert.Equal(t, h1.StreamID(), h.StreamID())
	assert.Equal(t, h1.Type(), h.Type())
	assert.Equal(t, h1.Flag(), h.Flag())

	fmt.Println("streamID:", h.StreamID())
	fmt.Println("type:", h.Type())
	fmt.Println("flag:", h)
}
