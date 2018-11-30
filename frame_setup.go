package rsocket

import (
	"encoding/binary"
	"time"
)

type FrameSetup struct {
	*Header
	major                uint16
	minor                uint16
	timeBetweenKeepalive time.Duration
	maxLifetime          time.Duration
	token                []byte
	mimeMetadata         string
	mimeData             string
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
	return p.timeBetweenKeepalive
}

func (p *FrameSetup) MaxLifetime() time.Duration {
	return p.maxLifetime
}

func (p *FrameSetup) Token() []byte {
	return p.token
}

func (p *FrameSetup) MetadataMIME() string {
	return p.mimeMetadata
}

func (p *FrameSetup) DataMIME() string {
	return p.mimeData
}

func (p *FrameSetup) Metadata() []byte {
	return p.metadata
}

func (p *FrameSetup) Data() []byte {
	return p.data
}

func (p *FrameSetup) Bytes() []byte {
	bs := p.Header.Bytes()
	v := make([]byte, 2)
	binary.BigEndian.PutUint16(v, p.major)
	bs = append(bs, v...)
	binary.BigEndian.PutUint16(v, p.minor)
	bs = append(bs, v...)
	t := make([]byte, 4)
	binary.BigEndian.PutUint32(t, uint32(p.timeBetweenKeepalive.Nanoseconds()/1000000))
	bs = append(bs, t...)
	binary.BigEndian.PutUint32(t, uint32(p.maxLifetime.Nanoseconds()/1000000))
	bs = append(bs, t...)
	// TODO: token if present
	bs = append(bs, byte(len(p.mimeMetadata)))
	bs = append(bs, p.mimeMetadata...)
	bs = append(bs, byte(len(p.mimeData)))
	bs = append(bs, p.mimeData...)
	if p.Header.Flags().Check(FlagMetadata) {
		bs = append(bs, encodeU24(len(p.metadata))...)
		bs = append(bs, p.metadata...)
	}
	bs = append(bs, p.data...)
	return bs
}

func asSetup(h *Header, raw []byte) *FrameSetup {
	var offset = frameHeaderLength
	major := binary.BigEndian.Uint16(raw[offset : offset+2])
	offset += 2
	minor := binary.BigEndian.Uint16(raw[offset : offset+2])
	offset += 2
	keepalive := binary.BigEndian.Uint32(raw[offset : offset+4])
	offset += 4
	maxLifetime := binary.BigEndian.Uint32(raw[offset : offset+4])
	offset += 4
	var token []byte
	if h.Flags().Check(FlagResume) {
		tokenLength := int(binary.BigEndian.Uint16(raw[offset : offset+2]))
		offset += 2
		token = raw[offset : offset+tokenLength]
		offset += tokenLength
	}
	mimeMetadataLen := int(raw[offset])
	offset += 1
	mimeMetadata := raw[offset : offset+mimeMetadataLen]
	offset += mimeMetadataLen
	mimeDataLen := int(raw[offset])
	offset += 1
	mimeData := raw[offset : offset+mimeDataLen]
	offset += mimeDataLen
	metadata, data := sliceMetadataAndData(h, raw, offset)
	return &FrameSetup{
		Header:               h,
		major:                major,
		minor:                minor,
		timeBetweenKeepalive: time.Millisecond * time.Duration(keepalive),
		maxLifetime:          time.Millisecond * time.Duration(maxLifetime),
		token:                token,
		mimeMetadata:         string(mimeMetadata),
		mimeData:             string(mimeData),
		metadata:             metadata,
		data:                 data,
	}
}

func mkSetup(meatadata []byte, data []byte, mimeMetadata string, mimeData string, f ...Flags) *FrameSetup {
	h := mkHeader(0, SETUP, f...)
	return &FrameSetup{
		Header:               h,
		metadata:             meatadata,
		data:                 data,
		mimeMetadata:         mimeMetadata,
		mimeData:             mimeData,
		major:                1,
		minor:                0,
		maxLifetime:          90 * time.Second,
		timeBetweenKeepalive: 30 * time.Second,
	}
}
