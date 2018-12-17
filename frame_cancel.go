package rsocket

import (
	"io"
)

type FrameCancel struct {
	*Header
}

func (p *FrameCancel) SetHeader(h *Header) {
	p.Header = h
}

func (p *FrameCancel) WriteTo(w io.Writer) (n int64, err error) {
	wrote, err := w.Write(p.Header.Bytes())
	n += int64(wrote)
	return
}

func (p *FrameCancel) Size() int {
	return headerLen
}

func (p *FrameCancel) Parse(h *Header, bs []byte) error {
	p.Header = h
	return nil
}

func mkCancel(sid uint32) *FrameCancel {
	return &FrameCancel{
		Header: mkHeader(sid, CANCEL),
	}
}
