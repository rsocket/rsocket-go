package protocol

import (
	"bufio"
)

type FrameDecoder struct {
	bf *bufio.Reader
}

type FramePayload struct {
}

type Frame struct {
	Length  int
	Payload *FramePayload
}

func (p *FrameDecoder) Read() (*Frame, error) {
	bs := make([]byte, 3)
	_, err := p.bf.Read(bs)
	if err != nil {
		return nil, err
	}
	panic("not implements")
}
