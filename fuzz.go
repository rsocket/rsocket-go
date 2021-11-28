//go:build gofuzz
// +build gofuzz

//go:generate GO111MODULE=off go-fuzz-build github.com/rsocket/rsocket-go/
package rsocket

import (
	"bytes"
	"errors"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
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
	var frame core.BufferedFrame
	frame, err = framing.FromBytes(raw)
	if err != nil {
		return
	}
	defer frame.Release()
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
