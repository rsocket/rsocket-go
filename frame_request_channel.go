package rsocket

import (
	"encoding/binary"
	"io"
)

type FrameRequestChannel struct {
	*Header
	initialRequestN uint32
	meatadata       []byte
	data            []byte
}

func (p *FrameRequestChannel) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int
	wrote, err = w.Write(p.Header.Bytes())
	n += int64(wrote)
	if err != nil {
		return
	}
	b4 := make([]byte, 4)
	binary.BigEndian.PutUint32(b4, p.initialRequestN)
	wrote, err = w.Write(b4)
	n += int64(wrote)
	if err != nil {
		return
	}

	if p.Header.Flags().Check(FlagMetadata) {
		wrote, err = w.Write(encodeU24(len(p.meatadata)))
		n += int64(wrote)
		if err != nil {
			return
		}
		wrote, err = w.Write(p.meatadata)
		n += int64(wrote)
		if err != nil {
			return
		}
	}
	if p.data == nil {
		return
	}
	wrote, err = w.Write(p.data)
	n += int64(wrote)
	return
}

func (p *FrameRequestChannel) Size() int {
	size := headerLen + 4
	if p.Header.Flags().Check(FlagMetadata) {
		size += 3 + len(p.meatadata)
	}
	if p.data != nil {
		size += len(p.data)
	}
	return size
}

func (p *FrameRequestChannel) InitialRequestN() uint32 {
	return p.initialRequestN
}

func (p *FrameRequestChannel) Metadata() []byte {
	return p.meatadata
}

func (p *FrameRequestChannel) Payload() []byte {
	return p.data
}

func (p *FrameRequestChannel) Parse(h *Header, bs []byte) error {
	p.Header = h
	p.meatadata, p.data = sliceMetadataAndData(p.Header, bs, headerLen+4)
	p.initialRequestN = binary.BigEndian.Uint32(bs[headerLen : headerLen+4])
	return nil
}

func mkRequestChannel(sid uint32, initalRequestN uint32, metadata []byte, data []byte, f ...Flags) *FrameRequestChannel {
	h := mkHeader(sid, REQUEST_CHANNEL, f...)
	return &FrameRequestChannel{
		Header:          h,
		initialRequestN: initalRequestN,
		meatadata:       metadata,
		data:            data,
	}
}
