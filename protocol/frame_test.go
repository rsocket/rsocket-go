package protocol

import (
	"encoding/hex"
	"log"
	"testing"
)

func TestName(t *testing.T) {
	f := NewFrameError(&BaseFrame{
		StreamID: 1,
		Type:     ERROR,
	}, ERR_REJECTED, []byte("foobar"))

	log.Println("streamID:", f.StreamID())
	log.Println("FlagIgnore:", f.IsIgnore())
	log.Println("FlagMetadata:", f.IsMetadata())
	log.Println("type:", f.Type())
}

func TestDecode(t *testing.T) {
	bs := []byte{
		0x00, 0x00, 0x00, 0x01, 0x28,
		0x60, 0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
	}

	f := Frame(bs)

	pf := &FramePayload{
		Frame: f,
	}

	log.Println("streamID:", f.StreamID())
	log.Println("type:", f.Type())
	log.Println("ignore:", f.IsIgnore())
	log.Println("isMetadata:", f.IsMetadata())
	log.Println("FlagFollow:", pf.IsFollow())
	log.Println("FlagFollow:", pf.IsComplete())
	log.Println("FlagNext:", pf.IsNext())
	log.Println("metadata:", string(pf.Metadata()))
	log.Println("payload:", string(pf.Payload()))

	f2 := NewPayload(&BaseFrame{
		Type:     PAYLOAD,
		Flags:    FlagComplete | FlagNext,
		StreamID: 1,
	}, []byte("foobar"), nil)
	log.Println(hex.EncodeToString(bs))
	log.Println(hex.EncodeToString(f2))
}
