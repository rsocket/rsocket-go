// +build gofuzz

//go:generate GO111MODULE=off go-fuzz-build github.com/rsocket/rsocket-go/

package rsocket

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/transport"
)

func Fuzz(data []byte) int {
	buf := bytes.NewBuffer(data)
	decoder := transport.NewLengthBasedFrameDecoder(buf)

	err := decoder.Handle(handleRaw)

	if err == nil || err == common.ErrInvalidFrame || err == transport.ErrIncompleteHeader {
		return 0
	}

	return 1
}

func handleRaw(raw []byte) (err error) {
	h := framing.ParseFrameHeader(raw)
	bf := common.BorrowByteBuffer()
	var frame framing.Frame
	frame, err = framing.NewFromBase(framing.NewBaseFrame(h, bf))
	if err != nil {
		return
	}

	// release resource borrowed
	defer frame.Release()

	switch f := frame.(type) {
	case fmt.Stringer:
		s := f.String()
		if len(s) > 0 {
			return
		}
	case error:
		e := f.Error()
		if len(e) > 0 {
			return
		}
	default:
		panic("unreachable")
	}

	return errors.New("???")
}
