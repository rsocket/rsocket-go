// +build gofuzz

//go:generate GO111MODULE=off go-fuzz-build github.com/rsocket/rsocket-go/
package rsocket

import (
	"bytes"
	"errors"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/common"
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
	return err == core.ErrInvalidFrame || err == transport.ErrIncompleteHeader
}

func handleRaw(raw []byte) (err error) {
	h := core.ParseFrameHeader(raw)
	bf := common.NewByteBuff()
	var frame core.Frame
	frame, err = framing.FromRawFrame(framing.NewRawFrame(h, bf))
	if err != nil {
		return
	}
	err = frame.Validate()
	if err != nil {
		return
	}
	if frame.Len() >= core.FrameHeaderLen {
		return
	}
	err = errors.New("broken frame")
	return
}
