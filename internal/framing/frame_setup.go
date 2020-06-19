package framing

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	_versionLen       = 4
	_timeLen          = 4
	_metadataLen      = 1
	_dataLen          = 1
	_minSetupFrameLen = _versionLen + _timeLen*2 + _metadataLen + _dataLen
)

// SetupFrame is sent by client to initiate protocol processing.
type SetupFrame struct {
	*RawFrame
}

// Validate returns error if frame is invalid.
func (p *SetupFrame) Validate() (err error) {
	if p.Len() < _minSetupFrameLen {
		err = errIncompleteFrame
	}
	return
}

// Version returns version.
func (p *SetupFrame) Version() common.Version {
	major := binary.BigEndian.Uint16(p.body.Bytes())
	minor := binary.BigEndian.Uint16(p.body.Bytes()[2:])
	return [2]uint16{major, minor}
}

// TimeBetweenKeepalive returns keepalive interval duration.
func (p *SetupFrame) TimeBetweenKeepalive() time.Duration {
	return time.Millisecond * time.Duration(binary.BigEndian.Uint32(p.body.Bytes()[4:]))
}

// MaxLifetime returns keepalive max lifetime.
func (p *SetupFrame) MaxLifetime() time.Duration {
	return time.Millisecond * time.Duration(binary.BigEndian.Uint32(p.body.Bytes()[8:]))
}

// Token returns token of setup.
func (p *SetupFrame) Token() []byte {
	if !p.header.Flag().Check(FlagResume) {
		return nil
	}
	raw := p.body.Bytes()
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
	if !p.header.Flag().Check(FlagMetadata) {
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
	if !p.header.Flag().Check(FlagMetadata) {
		return p.Body().Bytes()[offset:]
	}
	return p.trySliceData(offset)
}

// MetadataUTF8 returns metadata as UTF8 string
func (p *SetupFrame) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := p.Metadata()
	if ok {
		metadata = string(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (p *SetupFrame) DataUTF8() string {
	return string(p.Data())
}

func (p *SetupFrame) mime() (metadata []byte, data []byte) {
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

func (p *SetupFrame) seekMIME() int {
	if !p.header.Flag().Check(FlagResume) {
		return 12
	}
	l := binary.BigEndian.Uint16(p.body.Bytes()[12:])
	return 14 + int(l)
}

type SetupFrameSupport struct {
	*tinyFrame
	version      common.Version
	keepalive    [4]byte
	lifetime     [4]byte
	token        []byte
	mimeMetadata []byte
	mimeData     []byte
	metadata     []byte
	data         []byte
}

func (s SetupFrameSupport) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = s.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	wrote, err = s.version.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	var v int
	v, err = w.Write(s.keepalive[:])
	if err != nil {
		return
	}
	n += int64(v)

	v, err = w.Write(s.lifetime[:])
	if err != nil {
		return
	}
	n += int64(v)

	if s.header.Flag().Check(FlagResume) {
		tokenLen := len(s.token)
		err = binary.Write(w, binary.BigEndian, uint16(tokenLen))
		if err != nil {
			return
		}
		n += 2
		v, err = w.Write(s.token)
		if err != nil {
			return
		}
		n += int64(v)
	}

	lenMimeMetadata := len(s.mimeMetadata)
	v, err = w.Write([]byte{byte(lenMimeMetadata)})
	if err != nil {
		return
	}
	n += int64(v)
	v, err = w.Write(s.mimeMetadata)
	if err != nil {
		return
	}
	n += int64(v)

	lenMimeData := len(s.mimeData)
	v, err = w.Write([]byte{byte(lenMimeData)})
	if err != nil {
		return
	}
	n += int64(v)
	v, err = w.Write(s.mimeData)
	if err != nil {
		return
	}
	n += int64(v)

	wrote, err = writePayload(w, s.data, s.metadata)
	if err != nil {
		return
	}
	n += wrote
	return
}

func (s SetupFrameSupport) Len() int {
	n := _minSetupFrameLen + CalcPayloadFrameSize(s.data, s.metadata)
	n += len(s.mimeData) + len(s.mimeMetadata)
	if l := len(s.token); l > 0 {
		n += 2 + len(s.token)
	}
	return n
}

func NewSetupFrameSupport(
	version common.Version,
	timeBetweenKeepalive,
	maxLifetime time.Duration,
	token []byte,
	mimeMetadata []byte,
	mimeData []byte,
	data []byte,
	metadata []byte,
	lease bool,
) *SetupFrameSupport {
	var flag FrameFlag
	if l := len(token); l > 0 {
		flag |= FlagResume
	}
	if lease {
		flag |= FlagLease
	}
	if l := len(metadata); l > 0 {
		flag |= FlagMetadata
	}
	h := NewFrameHeader(0, FrameTypeSetup, flag)
	t := newTinyFrame(h)

	var a, b [4]byte
	binary.BigEndian.PutUint32(a[:], uint32(timeBetweenKeepalive.Nanoseconds()/1e6))
	binary.BigEndian.PutUint32(b[:], uint32(maxLifetime.Nanoseconds()/1e6))
	return &SetupFrameSupport{
		tinyFrame:    t,
		version:      version,
		keepalive:    a,
		lifetime:     b,
		token:        token,
		mimeMetadata: mimeMetadata,
		mimeData:     mimeData,
		metadata:     metadata,
		data:         data,
	}
}

// NewSetupFrame returns a new setup frame.
func NewSetupFrame(
	version common.Version,
	timeBetweenKeepalive,
	maxLifetime time.Duration,
	token []byte,
	mimeMetadata []byte,
	mimeData []byte,
	data []byte,
	metadata []byte,
	lease bool,
) *SetupFrame {
	var fg FrameFlag
	bf := common.NewByteBuff()
	if _, err := bf.Write(version.Bytes()); err != nil {
		panic(err)
	}
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], uint32(timeBetweenKeepalive.Nanoseconds()/1e6))
	if _, err := bf.Write(b4[:]); err != nil {
		panic(err)
	}
	binary.BigEndian.PutUint32(b4[:], uint32(maxLifetime.Nanoseconds()/1e6))
	if _, err := bf.Write(b4[:]); err != nil {
		panic(err)
	}
	if lease {
		fg |= FlagLease
	}
	if len(token) > 0 {
		fg |= FlagResume
		binary.BigEndian.PutUint16(b4[:2], uint16(len(token)))
		if _, err := bf.Write(b4[:2]); err != nil {
			panic(err)
		}
		if _, err := bf.Write(token); err != nil {
			panic(err)
		}
	}
	if err := bf.WriteByte(byte(len(mimeMetadata))); err != nil {
		panic(err)
	}
	if _, err := bf.Write(mimeMetadata); err != nil {
		panic(err)
	}
	if err := bf.WriteByte(byte(len(mimeData))); err != nil {
		panic(err)
	}
	if _, err := bf.Write(mimeData); err != nil {
		panic(err)
	}
	if len(metadata) > 0 {
		fg |= FlagMetadata
		if err := bf.WriteUint24(len(metadata)); err != nil {
			panic(err)
		}
		if _, err := bf.Write(metadata); err != nil {
			panic(err)
		}
	}
	if len(data) > 0 {
		if _, err := bf.Write(data); err != nil {
			panic(err)
		}
	}
	return &SetupFrame{
		NewRawFrame(NewFrameHeader(0, FrameTypeSetup, fg), bf),
	}
}
