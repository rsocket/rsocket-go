package rsocket

import (
	"encoding/binary"
	"fmt"
	"time"
)

type frameSetup struct {
	*baseFrame
}

func (p *frameSetup) String() string {
	return fmt.Sprintf(
		"frameSetup{%s,version=%s,kaInterval=%s,kaMaxLifetime=%s,token=%s,dataMimeType=%s,metadataMimeType=%s,data=%s,metadata=%s}",
		p.header, p.Version(), p.TimeBetweenKeepalive(), p.MaxLifetime(), p.Token(), p.DataMimeType(), p.MetadataMimeType(), p.Data(), p.Metadata(),
	)
}

func (p *frameSetup) Version() Version {
	major := binary.BigEndian.Uint16(p.body.Bytes())
	minor := binary.BigEndian.Uint16(p.body.Bytes()[2:])
	return [2]uint16{major, minor}
}

func (p *frameSetup) TimeBetweenKeepalive() time.Duration {
	return time.Millisecond * time.Duration(binary.BigEndian.Uint32(p.body.Bytes()[4:]))
}

func (p *frameSetup) MaxLifetime() time.Duration {
	return time.Millisecond * time.Duration(binary.BigEndian.Uint32(p.body.Bytes()[8:]))
}

func (p *frameSetup) Token() []byte {
	if !p.header.Flag().Check(FlagResume) {
		return nil
	}
	raw := p.body.Bytes()
	tokenLength := binary.BigEndian.Uint16(raw[12:])
	return raw[14 : 14+tokenLength]
}

func (p *frameSetup) DataMimeType() string {
	_, b := p.MIME()
	return string(b)
}

func (p *frameSetup) MetadataMimeType() string {
	a, _ := p.MIME()
	return string(a)
}

func (p *frameSetup) MIME() (metadata []byte, data []byte) {
	offset := p.seekMIME()
	raw := p.body.Bytes()
	l1 := int(raw[offset])
	offset += 1
	m1 := raw[offset : offset+l1]
	offset += l1
	l2 := int(raw[offset])
	offset += 1
	m2 := raw[offset : offset+l2]
	return m1, m2
}

func (p *frameSetup) Metadata() []byte {
	if !p.header.Flag().Check(FlagMetadata) {
		return nil
	}
	offset := p.seekMetadata()
	return p.trySliceMetadata(offset)
}

func (p *frameSetup) Data() []byte {
	offset := p.seekMetadata()
	return p.trySliceData(offset)
}

func (p *frameSetup) seekMIME() int {
	if !p.header.Flag().Check(FlagResume) {
		return 12
	}
	l := binary.BigEndian.Uint16(p.body.Bytes()[12:])
	return 14 + int(l)
}

func (p *frameSetup) seekMetadata() int {
	offset := p.seekMIME()
	m1, m2 := p.MIME()
	offset += 2 + len(m1) + len(m2)
	return offset
}

func createSetup(version Version, timeBetweenKeepalive, maxLifetime time.Duration, token, mimeMetadata, mimeData, data, metadata []byte) *frameSetup {
	var fg Flags
	bf := borrowByteBuffer()
	_, _ = bf.Write(version.Bytes())
	b4 := borrowByteBuffer()
	defer returnByteBuffer(b4)
	for i := 0; i < 4; i++ {
		_ = b4.WriteByte(0)
	}
	binary.BigEndian.PutUint32(b4.Bytes(), uint32(timeBetweenKeepalive.Nanoseconds()/1e6))
	_, _ = b4.WriteTo(bf)
	binary.BigEndian.PutUint32(b4.Bytes(), uint32(maxLifetime.Nanoseconds()/1e6))
	_, _ = b4.WriteTo(bf)
	if len(token) > 0 {
		fg |= FlagResume
		binary.BigEndian.PutUint16(b4.Bytes(), uint16(len(token)))
		_, _ = bf.Write(b4.Bytes()[:2])
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
	return &frameSetup{
		&baseFrame{
			header: createHeader(0, tSetup, fg),
			body:   bf,
		},
	}
}
