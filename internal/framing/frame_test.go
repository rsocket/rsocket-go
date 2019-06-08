package framing

import (
	"encoding/hex"
	"log"
	"testing"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestDecode_Payload(t *testing.T) {
	//s := "000000012940000005776f726c6468656c6c6f" // go
	s := "00000001296000000966726f6d5f6a617661706f6e67" //java
	bs, err := hex.DecodeString(s)
	assert.NoError(t, err, "bad bytes")
	h := ParseFrameHeader(bs[:HeaderLen])
	log.Println(h)
	bf := common.BorrowByteBuffer()
	_, _ = bf.Write(bs[HeaderLen:])
	pl := &FramePayload{
		BaseFrame: NewBaseFrame(h, bf),
	}
	defer pl.Release()
	log.Println(pl)
}
