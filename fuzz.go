// +build gofuzz

//go:generate GO111MODULE=off go-fuzz-build github.com/rsocket/rsocket-go/
package rsocket

import (
	"bytes"
	"errors"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/transport"
)

func Fuzz(data []byte) int {
	buf := bytes.NewBuffer(data)
	decoder := transport.NewLengthBasedFrameDecoder(buf)
	raw, err := decoder.Read()
	if err == nil {
		err = handleRaw(raw)
	}
	if err == nil || isExpectedError(err) {
		return 0
	}
	return 1
}

func isExpectedError(err error) bool {
	return err == common.ErrInvalidFrame || err == transport.ErrIncompleteHeader
}

func handleRaw(raw []byte) (err error) {
	h := framing.ParseFrameHeader(raw)
	bf := common.NewByteBuff()
	var frame framing.Frame
	frame, err = framing.FromRawFrame(framing.NewRawFrame(h, bf))
	if err != nil {
		return
	}
	err = frame.Validate()
	if err != nil {
		return
	}
	if frame.Len() >= framing.HeaderLen {
		return
	}
	err = errors.New("broken frame")
	return
}
