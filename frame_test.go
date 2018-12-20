package rsocket

import (
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestHeader_MarshallAndUnmarshall(t *testing.T) {
	h1 := mkHeader(134, PAYLOAD, FlagMetadata|FlagComplete|FlagNext)
	bs := h1.Bytes()
	h2 := &Header{}
	if err := h2.Parse(bs); err != nil {
		assert.NoError(t, err)
	}
	log.Printf("h1: %+v\n", h1)
	log.Printf("h2: %+v\n", h2)
	assert.Equal(t, h2.StreamID(), h1.StreamID(), "streamID doesn't matched")
	assert.Equal(t, h2.Type(), h1.Type(), "type doesn't matched")
	assert.Equal(t, h2.Flags(), h1.Flags(), "flags doesn't matched")
}
