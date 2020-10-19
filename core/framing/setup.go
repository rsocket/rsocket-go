package framing

import (
	"encoding/binary"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/bytesconv"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/u24"
)

const (
	_versionLen       = 4
	_timeLen          = 4
	_metadataLen      = 1
	_dataLen          = 1
	_minSetupFrameLen = _versionLen + _timeLen*2 + _metadataLen + _dataLen
)

// SetupFrame is Setup frame.
type SetupFrame struct {
	*bufferedFrame
}

// NewSetupFrame returns a new SetupFrame.
func NewSetupFrame(
	version core.Version,
	timeBetweenKeepalive,
	maxLifetime time.Duration,
	token []byte,
	mimeMetadata []byte,
	mimeData []byte,
	data []byte,
	metadata []byte,
	lease bool,
) *SetupFrame {
	var fg core.FrameFlag
	if lease {
		fg |= core.FlagLease
	}
	if len(token) > 0 {
		fg |= core.FlagResume
	}
	if len(metadata) > 0 {
		fg |= core.FlagMetadata
	}

	b := common.BorrowByteBuff()

	if err := core.WriteFrameHeader(b, 0, core.FrameTypeSetup, fg); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	if _, err := b.Write(version.Bytes()); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	if err := binary.Write(b, binary.BigEndian, uint32(common.ToMilliseconds(timeBetweenKeepalive))); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	if err := binary.Write(b, binary.BigEndian, uint32(common.ToMilliseconds(maxLifetime))); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	if len(token) > 0 {
		if err := binary.Write(b, binary.BigEndian, uint16(len(token))); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
		if _, err := b.Write(token); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
	}
	if err := b.WriteByte(byte(len(mimeMetadata))); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	if _, err := b.Write(mimeMetadata); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	if err := b.WriteByte(byte(len(mimeData))); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	if _, err := b.Write(mimeData); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	if len(metadata) > 0 {
		if err := u24.WriteUint24(b, len(metadata)); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
		if _, err := b.Write(metadata); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
	}
	if len(data) > 0 {
		if _, err := b.Write(data); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
	}

	return &SetupFrame{
		bufferedFrame: newBufferedFrame(b),
	}
}

// Validate returns error if frame is invalid.
func (p *SetupFrame) Validate() (err error) {
	if p.Len() < _minSetupFrameLen {
		err = errIncompleteFrame
	}
	return
}

// Version returns version.
func (p *SetupFrame) Version() core.Version {
	major := binary.BigEndian.Uint16(p.Body())
	minor := binary.BigEndian.Uint16(p.Body()[2:])
	return [2]uint16{major, minor}
}

// TimeBetweenKeepalive returns keepalive interval duration.
func (p *SetupFrame) TimeBetweenKeepalive() time.Duration {
	return time.Millisecond * time.Duration(binary.BigEndian.Uint32(p.Body()[4:]))
}

// MaxLifetime returns keepalive max lifetime.
func (p *SetupFrame) MaxLifetime() time.Duration {
	return time.Millisecond * time.Duration(binary.BigEndian.Uint32(p.Body()[8:]))
}

// Token returns token of setup.
func (p *SetupFrame) Token() []byte {
	if !p.HasFlag(core.FlagResume) {
		return nil
	}
	raw := p.Body()
	tokenLength := binary.BigEndian.Uint16(raw[12:])
	return raw[14 : 14+tokenLength]
}

// DataMimeType returns MIME of data.
func (p *SetupFrame) DataMimeType() (mime string) {
	_, b := p.mime()
	return string(b)
}

// MetadataMimeType returns MIME of metadata.
func (p *SetupFrame) MetadataMimeType() string {
	a, _ := p.mime()
	return string(a)
}

// Metadata returns metadata bytes.
func (p *SetupFrame) Metadata() ([]byte, bool) {
	if !p.HasFlag(core.FlagMetadata) {
		return nil, false
	}
	offset := p.seekMIME()
	m1, m2 := p.mime()
	offset += 2 + len(m1) + len(m2)
	return p.trySliceMetadata(offset)
}

// Data returns data bytes.
func (p *SetupFrame) Data() []byte {
	offset := p.seekMIME()
	m1, m2 := p.mime()
	offset += 2 + len(m1) + len(m2)
	if !p.HasFlag(core.FlagMetadata) {
		return p.Body()[offset:]
	}
	return p.trySliceData(offset)
}

// MetadataUTF8 returns metadata as UTF8 string
func (p *SetupFrame) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := p.Metadata()
	if ok {
		metadata = bytesconv.BytesToString(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (p *SetupFrame) DataUTF8() (data string) {
	b := p.Data()
	if len(b) > 0 {
		data = bytesconv.BytesToString(b)
	}
	return
}

func (p *SetupFrame) mime() (metadata []byte, data []byte) {
	offset := p.seekMIME()
	raw := p.Body()
	l1 := int(raw[offset])
	offset++
	m1 := raw[offset : offset+l1]
	offset += l1
	l2 := int(raw[offset])
	offset++
	m2 := raw[offset : offset+l2]
	return m1, m2
}

func (p *SetupFrame) seekMIME() int {
	if !p.HasFlag(core.FlagResume) {
		return 12
	}
	l := binary.BigEndian.Uint16(p.Body()[12:])
	return 14 + int(l)
}
