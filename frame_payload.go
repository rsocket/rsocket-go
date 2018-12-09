package rsocket

import (
	"io"
)

type FramePayload struct {
	*Header
	metadata []byte
	data     []byte
}

func (p *FramePayload) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int
	wrote, err = w.Write(p.Header.Bytes())
	n += int64(wrote)
	if err != nil {
		return
	}
	if p.Header.Flags().Check(FlagMetadata) {
		wrote, err = w.Write(encodeU24(len(p.metadata)))
		n += int64(wrote)
		if err != nil {
			return
		}
		wrote, err = w.Write(p.metadata)
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

func (p *FramePayload) Size() int {
	size := headerLen
	if p.Header.Flags().Check(FlagMetadata) {
		size += 3 + len(p.metadata)
	}
	if p.data != nil {
		size += len(p.data)
	}
	return size
}

func (p *FramePayload) Metadata() []byte {
	return p.metadata
}

func (p *FramePayload) Data() []byte {
	return p.data
}

func asPayload(h *Header, raw []byte) *FramePayload {
	m, d := sliceMetadataAndData(h, raw, headerLen)
	return &FramePayload{
		Header:   h,
		metadata: m,
		data:     d,
	}
}

func mkPayload(sid uint32, metadata []byte, data []byte, f ...Flags) *FramePayload {
	return &FramePayload{
		Header:   mkHeader(sid, PAYLOAD, f...),
		metadata: metadata,
		data:     data,
	}
}
