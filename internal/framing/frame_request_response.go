package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/internal/common"
)

// RequestResponseFrame is frame for requesting single response.
type RequestResponseFrame struct {
	*RawFrame
}

type RequestResponseFrameSupport struct {
	*tinyFrame
	metadata []byte
	data     []byte
}

// Validate returns error if frame is invalid.
func (r *RequestResponseFrame) Validate() (err error) {
	if r.header.Flag().Check(FlagMetadata) && r.body.Len() < 3 {
		err = errIncompleteFrame
	}
	return
}

// Metadata returns metadata bytes.
func (r *RequestResponseFrame) Metadata() ([]byte, bool) {
	return r.trySliceMetadata(0)
}

// Data returns data bytes.
func (r *RequestResponseFrame) Data() []byte {
	return r.trySliceData(0)
}

// MetadataUTF8 returns metadata as UTF8 string.
func (r *RequestResponseFrame) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := r.Metadata()
	if ok {
		metadata = string(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (r *RequestResponseFrame) DataUTF8() string {
	return string(r.Data())
}

func (r RequestResponseFrameSupport) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = r.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote
	wrote, err = writePayload(w, r.data, r.metadata)
	if err == nil {
		n += wrote
	}
	return
}

func (r RequestResponseFrameSupport) Len() int {
	return CalcPayloadFrameSize(r.data, r.metadata)
}

// NewRequestResponseFrameSupport returns a new RequestResponse frame support.
func NewRequestResponseFrameSupport(id uint32, data, metadata []byte, fg FrameFlag) FrameSupport {
	if len(metadata) > 0 {
		fg |= FlagMetadata
	}
	return &RequestResponseFrameSupport{
		tinyFrame: newTinyFrame(NewFrameHeader(id, FrameTypeRequestResponse, fg)),
		metadata:  metadata,
		data:      data,
	}
}

// NewRequestResponseFrame returns a new RequestResponse frame.
func NewRequestResponseFrame(id uint32, data, metadata []byte, fg FrameFlag) *RequestResponseFrame {
	bf := common.NewByteBuff()
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
	return &RequestResponseFrame{
		NewRawFrame(NewFrameHeader(id, FrameTypeRequestResponse, fg), bf),
	}
}
