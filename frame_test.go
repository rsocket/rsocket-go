package rsocket

import (
	"log"
	"testing"
)

func TestHeader_MarshallAndUnmarshall(t *testing.T) {
	h1 := mkHeader(134, PAYLOAD, FlagMetadata|FlagComplete|FlagNext)
	bs := h1.Bytes()
	h2, err := asHeader(bs)
	if err != nil {
		t.Error(err)
	}
	log.Printf("h1: %+v\n", h1)
	log.Printf("h2: %+v\n", h2)
	if h1.StreamID() != h2.StreamID() {
		t.Errorf("streamID doesn't matched")
	}
	if h1.Type() != h2.Type() {
		t.Errorf("type doesn't matched")
	}
	if h1.Flags() != h2.Flags() {
		t.Errorf("flags doesn't matched")
	}
}
