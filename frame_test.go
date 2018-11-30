package rsocket

import (
	"log"
	"testing"
)

func TestHeader_DEC(t *testing.T) {
	h := mkHeader(134, PAYLOAD, FlagMetadata|FlagComplete|FlagNext)
	bs := h.Bytes()
	h2, err := asHeader(bs)
	if err != nil {
		t.Error(err)
	}
	log.Printf("h1: %+v\n", h)
	log.Printf("h2: %+v\n", h2)
}

func TestName(t *testing.T) {
	foo := mkPayload(1, []byte("mmm"), []byte("ddd"), FlagNext, FlagComplete, FlagMetadata)
	raw := foo.Bytes()

	h, err := asHeader(raw)
	if err != nil {
		t.Error(err)
	}

	bar := asPayload(h, raw)

	log.Printf("%+v\n", foo)
	log.Printf("%+v\n", bar)

}
