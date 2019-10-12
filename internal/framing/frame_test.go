package framing

import (
	"encoding/hex"
	"log"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestDecode_Payload(t *testing.T) {
	//s := "000000012940000005776f726c6468656c6c6f" // go
	//s := "00000001296000000966726f6d5f6a617661706f6e67" //java

	var all []string
	all = append(all, "0000000004400001000000004e2000015f90126170706c69636174696f6e2f62696e617279126170706c69636174696f6e2f62696e617279")
	all = append(all, "00000000090000000bb800000005")
	all = append(all, "00000000090000001b5800000005")
	all = append(all, "000000011100000000436c69656e74207265717565737420547565204f63742032322032303a31373a3333204353542032303139")
	all = append(all, "00000001286053657276657220526573706f6e736520547565204f63742032322032303a31373a3333204353542032303139")

	for _, s := range all {
		bs, err := hex.DecodeString(s)
		assert.NoError(t, err, "bad bytes")
		h := ParseFrameHeader(bs[:HeaderLen])
		//log.Println(h)
		bf := common.NewByteBuff()
		_, _ = bf.Write(bs[HeaderLen:])
		f, err := NewFromBase(NewBaseFrame(h, bf))
		assert.NoError(t, err, "decode failed")
		log.Println(f)
	}

	lease := NewFrameLease(3*time.Second, 5, nil)
	log.Println("actual:", hex.EncodeToString(lease.Bytes()))
	log.Println("should: 00000000090000000bb800000005")
}
