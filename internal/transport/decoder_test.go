package transport

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
)

func TestDecoder(t *testing.T) {
	bs, _ := hex.DecodeString("000012000000012920000003797979776f726c6432000006000000012840")
	r := bytes.NewBuffer(bs)

	d := NewLengthBasedFrameDecoder(r)

	for {
		raw, err := d.Read()
		if err != nil {
			break
		}
		h := framing.ParseFrameHeader(raw)
		bf := common.BorrowByteBuffer()
		_, _ = bf.Write(raw[framing.HeaderLen:])
		f, err := framing.NewFromBase(framing.NewBaseFrame(h, bf))
		if err != nil {
			panic(err)
		}
		fmt.Println(f)
	}

}
