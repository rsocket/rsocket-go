package rsocket

import (
	"encoding/binary"
	"io"
)

type FrameRequestN struct {
	*Header
	n uint32
}

func (p *FrameRequestN) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int
	wrote, err = w.Write(p.Header.Bytes())
	n += int64(wrote)
	if err != nil {
		return
	}
	b4 := make([]byte, 4)
	binary.BigEndian.PutUint32(b4, p.n)
	wrote, err = w.Write(b4)
	n += int64(wrote)
	return
}

func (p *FrameRequestN) Size() int {
	return headerLen + 4
}

func (p *FrameRequestN) RequestN() uint32 {
	return p.n
}

func (p *FrameRequestN) Parse(h *Header, bs []byte) error {
	p.Header = h
	p.n = binary.BigEndian.Uint32(bs[headerLen : headerLen+4])
	return nil
}

func mkRequestN(sid uint32, n uint32, f ...Flags) *FrameRequestN {
	h := mkHeader(sid, REQUEST_N, f...)
	return &FrameRequestN{
		Header: h,
		n:      n,
	}
}
