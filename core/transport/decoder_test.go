package transport

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/internal/common"
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
		h := core.ParseFrameHeader(raw)
		bf := common.NewByteBuff()
		_, _ = bf.Write(raw[core.FrameHeaderLen:])
		f, err := framing.FromRawFrame(framing.NewRawFrame(h, bf))
		if err != nil {
			panic(err)
		}
		fmt.Println(f)
	}

}
