package rsocket

import "io"

type FrameMetadataPush struct {
	*Header
	metadata []byte
}

func (p *FrameMetadataPush) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int
	wrote, err = w.Write(p.Header.Bytes())
	n += int64(wrote)
	if err != nil {
		return
	}
	wrote, err = w.Write(p.metadata)
	n += int64(wrote)
	return
}

func (p *FrameMetadataPush) Size() int {
	return headerLen + len(p.metadata)
}

func (p *FrameMetadataPush) Metadata() []byte {
	return p.metadata
}

func (p *FrameMetadataPush) Parse(h *Header, bs []byte) error {
	p.Header = h
	m := bs[headerLen:]
	clone := make([]byte, len(m))
	copy(clone, m)
	p.metadata = clone
	return nil
}

func mkMetadataPush(sid uint32, metadata []byte, f ...Flags) *FrameMetadataPush {
	return &FrameMetadataPush{
		Header:   mkHeader(sid, METADATA_PUSH, f...),
		metadata: metadata,
	}
}
