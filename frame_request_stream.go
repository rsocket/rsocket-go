package rsocket

import (
	"encoding/binary"
	"io"
)

type FrameRequestStream struct {
	*Header
	initialRequestN uint32
	metadata        []byte
	data            []byte
}

func (p FrameRequestStream) WriteTo(w io.Writer) (n int64, err error) {
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

func (p FrameRequestStream) Size() int {
	size := headerLen + 4
	if p.Header.Flags().Check(FlagMetadata) {
		size += 3 + len(p.metadata)
	}
	if p.data != nil {
		size += len(p.data)
	}
	return size
}

func (p FrameRequestStream) InitialRequestN() uint32 {
	return p.initialRequestN
}

func (p FrameRequestStream) Metadata() []byte {
	return p.metadata
}

func (p FrameRequestStream) Data() []byte {
	return p.data
}

func asRequestStream(h *Header, raw []byte) *FrameRequestStream {
	m, d := sliceMetadataAndData(h, raw, headerLen+4)
	return &FrameRequestStream{
		Header:          h,
		initialRequestN: binary.BigEndian.Uint32(raw[headerLen : headerLen+4]),
		metadata:        m,
		data:            d,
	}
}

func mkRequestStream(sid uint32, n uint32, metadata []byte, data []byte, f ...Flags) *FrameRequestStream {
	return &FrameRequestStream{
		Header:          mkHeader(sid, REQUEST_STREAM, f...),
		initialRequestN: n,
		metadata:        metadata,
		data:            data,
	}
}
