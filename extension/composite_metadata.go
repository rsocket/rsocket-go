package extension

import (
	"fmt"
	"io"
	"math"

	"github.com/rsocket/rsocket-go/common"
)

// CompositeMetadata provides multi Metadata payloads with different MIME types.
type CompositeMetadata interface {
	io.WriterTo
	// MIME returns MIME type.
	MIME() string
	// Payload returns bytes of Metadata payload.
	Payload() []byte

	encode() ([]byte, error)
}

// DecodeCompositeMetadata decode bytes to composite metadata.
func DecodeCompositeMetadata(raw []byte) ([]CompositeMetadata, error) {
	ret := make([]CompositeMetadata, 0)
	offset := 0
	for offset < len(raw) {
		l, cm, err := decodeCompositeMetadataOnce(raw[offset:])
		if err != nil {
			return nil, err
		}
		ret = append(ret, cm)
		offset += l
	}
	return ret, nil
}

// NewCompositeMetadata returns a new composite metadata.
func NewCompositeMetadata(mime string, metadataPayload []byte) CompositeMetadata {
	if found, ok := mimeTypesR[mime]; ok {
		return &implCompositeMetadata{
			mime: found,
			data: metadataPayload,
		}
	}
	return &implCompositeMetadata{
		mime: mime,
		data: metadataPayload,
	}
}

func decodeCompositeMetadataOnce(raw []byte) (l int, cm CompositeMetadata, err error) {
	m := raw[0]
	size := 1
	idOrLen := (m << 1) >> 1
	ret := &implCompositeMetadata{}
	if m&0x80 == 0x80 {
		ret.mime = MIME(idOrLen)
	} else {
		size += int(idOrLen)
		ret.mime = string(raw[1 : 1+idOrLen])
	}
	metadataLen := common.NewUint24Bytes(raw[size:size+3]).AsInt()
	end := size + 3 + metadataLen
	ret.data = raw[size+3 : end]
	return end, ret, nil
}

type implCompositeMetadata struct {
	mime interface{}
	data []byte
}

func (p *implCompositeMetadata) WriteTo(w io.Writer) (n int64, err error) {
	var bs []byte
	bs, err = p.encode()
	if err != nil {
		return
	}
	var wrote int
	wrote, err = w.Write(bs)
	if err != nil {
		return
	}
	n = int64(wrote)
	return
}

func (p *implCompositeMetadata) String() string {
	return fmt.Sprintf("CompositeMetadata{MIME=%s,payload=%s}", p.MIME(), p.Payload())
}

func (p *implCompositeMetadata) MIME() string {
	switch mime := p.mime.(type) {
	case MIME:
		found, ok := mimeTypes[mime]
		if !ok {
			panic(fmt.Errorf("invalid MIME ID: %d", mime))
		}
		return found
	case string:
		return mime
	}
	panic("unreachable")
}

func (p *implCompositeMetadata) Payload() []byte {
	return p.data
}

func (p *implCompositeMetadata) encode() ([]byte, error) {
	bs := make([]byte, 0)
	switch v := p.mime.(type) {
	case MIME:
		bs = append(bs, 0x80|byte(v))
	case string:
		l := len(v)
		if l > math.MaxInt8 {
			return nil, fmt.Errorf("length of MIME type is over %d", math.MaxInt8)
		}
		bs = append(bs, byte(l))
		bs = append(bs, []byte(v)...)
	default:
		panic("unreachable")
	}
	bs = append(bs, common.NewUint24(len(p.data)).Bytes()...)
	bs = append(bs, p.data...)
	return bs, nil
}
