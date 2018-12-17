package rsocket

import (
	"encoding/binary"
	"io"
	"time"
)

type FrameSetup struct {
	*Header
	major                uint16
	minor                uint16
	timeBetweenKeepalive uint32
	maxLifetime          uint32
	token                []byte
	mimeMetadata         []byte
	mimeData             []byte
	metadata             []byte
	data                 []byte
}

func (p *FrameSetup) Major() uint16 {
	return p.major
}

func (p *FrameSetup) Minor() uint16 {
	return p.minor
}

func (p *FrameSetup) TimeBetweenKeepalive() time.Duration {
	return time.Millisecond * time.Duration(p.timeBetweenKeepalive)
}

func (p *FrameSetup) MaxLifetime() time.Duration {
	return time.Millisecond * time.Duration(p.maxLifetime)
}

func (p *FrameSetup) Token() []byte {
	return p.token
}

func (p *FrameSetup) MetadataMIME() []byte {
	return p.mimeMetadata
}

func (p *FrameSetup) DataMIME() []byte {
	return p.mimeData
}

func (p *FrameSetup) Metadata() []byte {
	return p.metadata
}

func (p *FrameSetup) Data() []byte {
	return p.data
}

func (p *FrameSetup) Size() int {
	size := headerLen + 12
	if p.Header.Flags().Check(FlagResume) {
		size += 2
		size += len(p.token)
	}
	size += 1 + len(p.mimeMetadata)
	size += 1 + len(p.mimeData)
	if p.Header.Flags().Check(FlagMetadata) {
		size += 3 + len(p.metadata)
	}
	if p.data != nil {
		size += len(p.data)
	}
	return size
}

func (p *FrameSetup) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int
	wrote, err = w.Write(p.Header.Bytes())
	n += int64(wrote)
	if err != nil {
		return
	}
	b2 := make([]byte, 2)
	binary.BigEndian.PutUint16(b2, p.major)
	wrote, err = w.Write(b2)
	n += int64(wrote)
	if err != nil {
		return
	}
	binary.BigEndian.PutUint16(b2, p.minor)
	wrote, err = w.Write(b2)
	n += int64(wrote)
	if err != nil {
		return
	}
	b4 := make([]byte, 4)
	binary.BigEndian.PutUint32(b4, p.timeBetweenKeepalive)
	wrote, err = w.Write(b4)
	n += int64(wrote)
	if err != nil {
		return
	}
	binary.BigEndian.PutUint32(b4, p.maxLifetime)
	wrote, err = w.Write(b4)
	n += int64(wrote)
	if err != nil {
		return
	}
	if p.Header.Flags().Check(FlagResume) {
		binary.BigEndian.PutUint16(b2, uint16(len(p.token)))
		wrote, err = w.Write(b2)
		n += int64(wrote)
		if err != nil {
			return
		}
		wrote, err = w.Write(p.token)
		n += int64(wrote)
		if err != nil {
			return
		}
	}
	b := make([]byte, 1)
	b[0] = byte(len(p.mimeMetadata))
	wrote, err = w.Write(b)
	n += int64(wrote)
	if err != nil {
		return
	}
	wrote, err = w.Write(p.mimeMetadata)
	n += int64(wrote)
	if err != nil {
		return
	}
	b[0] = byte(len(p.mimeData))
	wrote, err = w.Write(b)
	n += int64(wrote)
	if err != nil {
		return
	}
	wrote, err = w.Write(p.mimeData)
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

func (p *FrameSetup) Parse(h *Header, raw []byte) error {
	p.Header = h
	var offset = headerLen
	p.major = binary.BigEndian.Uint16(raw[offset : offset+2])
	offset += 2
	p.minor = binary.BigEndian.Uint16(raw[offset : offset+2])
	offset += 2
	p.timeBetweenKeepalive = binary.BigEndian.Uint32(raw[offset : offset+4])
	offset += 4
	p.maxLifetime = binary.BigEndian.Uint32(raw[offset : offset+4])
	offset += 4
	if p.Header.Flags().Check(FlagResume) {
		tokenLength := int(binary.BigEndian.Uint16(raw[offset : offset+2]))
		offset += 2
		p.token = make([]byte, tokenLength)
		copy(p.token, raw[offset:offset+tokenLength])
		offset += tokenLength
	} else {
		p.token = nil
	}
	mimeMetadataLen := int(raw[offset])
	offset += 1
	p.mimeMetadata = make([]byte, mimeMetadataLen)
	copy(p.mimeMetadata, raw[offset:offset+mimeMetadataLen])
	offset += mimeMetadataLen
	mimeDataLen := int(raw[offset])
	offset += 1
	p.mimeData = make([]byte, mimeDataLen)
	copy(p.mimeData, raw[offset:offset+mimeDataLen])
	offset += mimeDataLen
	p.metadata, p.data = sliceMetadataAndData(p.Header, raw, offset)
	return nil
}

func mkSetup(meatadata []byte, data []byte, mimeMetadata []byte, mimeData []byte, f ...Flags) *FrameSetup {
	h := mkHeader(0, SETUP, f...)
	return &FrameSetup{
		Header:               h,
		metadata:             meatadata,
		data:                 data,
		mimeMetadata:         mimeMetadata,
		mimeData:             mimeData,
		major:                1,
		minor:                0,
		maxLifetime:          90000,
		timeBetweenKeepalive: 30000,
	}
}
