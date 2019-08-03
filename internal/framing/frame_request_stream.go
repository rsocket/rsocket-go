package framing

import (
	"encoding/binary"
	"fmt"

	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	minRequestStreamFrameLen = initReqLen
)

// FrameRequestStream is frame for requesting a completable stream.
type FrameRequestStream struct {
	*BaseFrame
}

// Validate returns error if frame is invalid.
func (p *FrameRequestStream) Validate() (err error) {
	if p.body.Len() < minRequestStreamFrameLen {
		err = errIncompleteFrame
	}
	return
}

func (p *FrameRequestStream) String() string {
	m, _ := p.MetadataUTF8()
	return fmt.Sprintf("FrameRequestStream{%s,data=%s,metadata=%s,initialRequestN=%d}",
		p.header, p.DataUTF8(), m, p.InitialRequestN())
}

// InitialRequestN returns initial request N.
func (p *FrameRequestStream) InitialRequestN() uint32 {
	return binary.BigEndian.Uint32(p.body.Bytes())
}

// Metadata returns metadata bytes.
func (p *FrameRequestStream) Metadata() ([]byte, bool) {
	return p.trySliceMetadata(4)
}

// Data returns data bytes.
func (p *FrameRequestStream) Data() []byte {
	return p.trySliceData(4)
}

// MetadataUTF8 returns metadata as UTF8 string.
func (p *FrameRequestStream) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := p.Metadata()
	if ok {
		metadata = common.Bytes2str(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (p *FrameRequestStream) DataUTF8() string {
	return common.Bytes2str(p.Data())
}

// NewFrameRequestStream returns a new request stream frame.
func NewFrameRequestStream(id uint32, n uint32, data, metadata []byte, flags ...FrameFlag) *FrameRequestStream {
	fg := newFlags(flags...)
	bf := common.New()
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], n)
	if _, err := bf.Write(b4[:]); err != nil {
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
	return &FrameRequestStream{
		NewBaseFrame(NewFrameHeader(id, FrameTypeRequestStream, fg), bf),
	}
}
