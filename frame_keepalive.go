package rsocket

import (
	"encoding/binary"
	"io"
)

type FrameKeepalive struct {
	*Header
	lastReceivedPosition uint64
	data                 []byte
}

func (p *FrameKeepalive) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int
	wrote, err = w.Write(p.Header.Bytes())
	n += int64(wrote)
	if err != nil {
		return
	}
	b8 := make([]byte, 8)
	binary.BigEndian.PutUint64(b8, p.lastReceivedPosition)
	wrote, err = w.Write(b8)
	n += int64(wrote)
	if err != nil {
		return
	}
	if p.data == nil {
		return
	}
	wrote, err = w.Write(p.data)
	n += int64(wrote)
	return
}

func (p *FrameKeepalive) Size() int {
	size := headerLen + 8
	if p.data != nil {
		size += len(p.data)
	}
	return size
}

func (p *FrameKeepalive) LastReceivedPosition() uint64 {
	return p.lastReceivedPosition
}

func (p *FrameKeepalive) Data() []byte {
	return p.data
}

func (p *FrameKeepalive) Parse(h *Header, bs []byte) error {
	p.Header = h
	p.lastReceivedPosition = binary.BigEndian.Uint64(bs[headerLen : headerLen+8])
	data := bs[headerLen+8:]
	p.data = make([]byte, len(data))
	copy(p.data, data)
	return nil
}

func mkKeepalive(sid uint32, pos uint64, data []byte, f ...Flags) *FrameKeepalive {
	h := mkHeader(sid, KEEPALIVE, f...)
	return &FrameKeepalive{
		Header:               h,
		lastReceivedPosition: pos,
		data:                 data,
	}
}
