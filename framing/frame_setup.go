package framing

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/rsocket/rsocket-go/common"
)

const (
	versionLen       = 4
	timeLen          = 4
	tokenLen         = 2
	metadataLen      = 1
	dataLen          = 1
	minSetupFrameLen = versionLen + timeLen*2 + tokenLen + metadataLen + dataLen
)

// FrameSetup is sent by client to initiate protocol processing.
type FrameSetup struct {
	*BaseFrame
}

// Validate returns error if frame is invalid.
func (p *FrameSetup) Validate() (err error) {
	if p.Len() < minSetupFrameLen {
		err = errIncompleteFrame
	}
	return
}

func (p *FrameSetup) String() string {
	m, _ := p.MetadataUTF8()
	return fmt.Sprintf(
		"FrameSetup{%s,version=%s,keepaliveInterval=%s,keepaliveMaxLifetime=%s,token=%s,dataMimeType=%s,metadataMimeType=%s,data=%s,metadata=%s}",
		p.header, p.Version(), p.TimeBetweenKeepalive(), p.MaxLifetime(), p.Token(), p.DataMimeType(), p.MetadataMimeType(), p.DataUTF8(), m,
	)
}

// Version returns version.
func (p *FrameSetup) Version() common.Version {
	major := binary.BigEndian.Uint16(p.body.Bytes())
	minor := binary.BigEndian.Uint16(p.body.Bytes()[2:])
	return [2]uint16{major, minor}
}

// TimeBetweenKeepalive returns keepalive interval duration.
func (p *FrameSetup) TimeBetweenKeepalive() time.Duration {
	return time.Millisecond * time.Duration(binary.BigEndian.Uint32(p.body.Bytes()[4:]))
}

// MaxLifetime returns keepalive max lifetime.
func (p *FrameSetup) MaxLifetime() time.Duration {
	return time.Millisecond * time.Duration(binary.BigEndian.Uint32(p.body.Bytes()[8:]))
}

// Token returns token of setup.
func (p *FrameSetup) Token() []byte {
	if !p.header.Flag().Check(FlagResume) {
		return nil
	}
	raw := p.body.Bytes()
	tokenLength := binary.BigEndian.Uint16(raw[12:])
	return raw[14 : 14+tokenLength]
}

// DataMimeType returns MIME of data.
func (p *FrameSetup) DataMimeType() string {
	_, b := p.mime()
	return string(b)
}

// MetadataMimeType returns MIME of metadata.
func (p *FrameSetup) MetadataMimeType() string {
	a, _ := p.mime()
	return string(a)
}

// Metadata returns metadata bytes.
func (p *FrameSetup) Metadata() ([]byte, bool) {
	if !p.header.Flag().Check(FlagMetadata) {
		return nil, false
	}
	offset := p.seekMetadata()
	return p.trySliceMetadata(offset)
}

// Data returns data bytes.
func (p *FrameSetup) Data() []byte {
	offset := p.seekMetadata()
	return p.trySliceData(offset)
}

// MetadataUTF8 returns metadata as UTF8 string
func (p *FrameSetup) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := p.Metadata()
	if ok {
		metadata = string(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (p *FrameSetup) DataUTF8() string {
	return string(p.Data())
}

func (p *FrameSetup) mime() (metadata []byte, data []byte) {
	offset := p.seekMIME()
	raw := p.body.Bytes()
	l1 := int(raw[offset])
	offset++
	m1 := raw[offset : offset+l1]
	offset += l1
	l2 := int(raw[offset])
	offset++
	m2 := raw[offset : offset+l2]
	return m1, m2
}

func (p *FrameSetup) seekMIME() int {
	if !p.header.Flag().Check(FlagResume) {
		return 12
	}
	l := binary.BigEndian.Uint16(p.body.Bytes()[12:])
	return 14 + int(l)
}

func (p *FrameSetup) seekMetadata() int {
	offset := p.seekMIME()
	m1, m2 := p.mime()
	offset += 2 + len(m1) + len(m2)
	return offset
}

// NewFrameSetup returns a new setup frame.
func NewFrameSetup(version common.Version, timeBetweenKeepalive, maxLifetime time.Duration, token, mimeMetadata, mimeData, data, metadata []byte) *FrameSetup {
	var fg FrameFlag
	bf := common.BorrowByteBuffer()
	_, _ = bf.Write(version.Bytes())
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], uint32(timeBetweenKeepalive.Nanoseconds()/1e6))
	_, _ = bf.Write(b4[:])
	binary.BigEndian.PutUint32(b4[:], uint32(maxLifetime.Nanoseconds()/1e6))
	_, _ = bf.Write(b4[:])
	if len(token) > 0 {
		fg |= FlagResume
		binary.BigEndian.PutUint16(b4[:2], uint16(len(token)))
		_, _ = bf.Write(b4[:2])
		_, _ = bf.Write(token)
	}
	_ = bf.WriteByte(byte(len(mimeMetadata)))
	_, _ = bf.Write(mimeMetadata)
	_ = bf.WriteByte(byte(len(mimeData)))
	_, _ = bf.Write(mimeData)
	if len(metadata) > 0 {
		fg |= FlagMetadata
		_ = bf.WriteUint24(len(metadata))
		_, _ = bf.Write(metadata)
	}
	if len(data) > 0 {
		_, _ = bf.Write(data)
	}
	return &FrameSetup{
		&BaseFrame{
			header: NewFrameHeader(0, FrameTypeSetup, fg),
			body:   bf,
		},
	}
}
