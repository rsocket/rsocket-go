package extension

import (
	"bytes"
	"fmt"
	"math"

	"github.com/rsocket/rsocket-go/internal/common"
)

// CompositeMetadata provides multi Metadata payloads with different MIME types.
type CompositeMetadata []byte

// CompositeMetadataScanner can be used to scan entry in CompositeMetadata.
type CompositeMetadataScanner struct {
	offset int
	raw    []byte
}

// CompositeMetadataBuilder can be used to build a CompositeMetadata.
type CompositeMetadataBuilder struct {
	k []interface{}
	v [][]byte
}

// Scanner returns a entry scanner.
func (c CompositeMetadata) Scanner() *CompositeMetadataScanner {
	return &CompositeMetadataScanner{
		offset: 0,
		raw:    c,
	}
}

// Scan returns true when scanner has more content.
func (c *CompositeMetadataScanner) Scan() bool {
	return c.offset < len(c.raw)
}

// MetadataUTF8 returns current metadata in utf8 string.
func (c *CompositeMetadataScanner) MetadataUTF8() (mimeType string, metadata string, err error) {
	mimeType, metadataRaw, err := c.Metadata()
	if err != nil {
		return
	}
	metadata = string(metadataRaw)
	return
}

// MetadataUTF8 returns current metadata bytes.
func (c *CompositeMetadataScanner) Metadata() (mimeType string, metadata []byte, err error) {
	l, mimeType, metadata, err := c.decodeCompositeMetadataOnce(c.raw[c.offset:])
	if err != nil {
		return
	}
	c.offset += l
	return
}

func (c *CompositeMetadataScanner) decodeCompositeMetadataOnce(raw []byte) (length int, mimeType string, metadata []byte, err error) {
	m := raw[0]
	size := 1
	idOrLen := (m << 1) >> 1
	if m&0x80 == 0x80 {
		mimeType = MIME(idOrLen).String()
	} else {
		mimeTypeLen := int(idOrLen) + 1
		size += mimeTypeLen
		mimeType = string(raw[1 : 1+mimeTypeLen])
	}
	metadataLen := common.NewUint24Bytes(raw[size : size+3]).AsInt()
	length = size + 3 + metadataLen
	metadata = raw[size+3 : length]
	return
}

// PushWellKnown push a WellKnownMimeType and metadata bytes.
func (c *CompositeMetadataBuilder) PushWellKnown(mimeType MIME, metadata []byte) *CompositeMetadataBuilder {
	c.k = append(c.k, mimeType)
	c.v = append(c.v, metadata)
	return c
}

// PushWellKnownString push a WellKnownMimeType and metadata string.
func (c *CompositeMetadataBuilder) PushWellKnownString(mimeType MIME, metadata string) *CompositeMetadataBuilder {
	return c.PushWellKnown(mimeType, []byte(metadata))
}

// Push push a custom MimeType and metadata bytes.
func (c *CompositeMetadataBuilder) Push(mimeType string, metadata []byte) *CompositeMetadataBuilder {
	if well, ok := ParseMIME(mimeType); ok {
		c.k = append(c.k, well)
	} else {
		c.k = append(c.k, mimeType)
	}
	c.v = append(c.v, metadata)
	return c
}

// PushString push a custom MimeType and metadata string.
func (c *CompositeMetadataBuilder) PushString(mimeType string, metadata string) *CompositeMetadataBuilder {
	return c.Push(mimeType, []byte(metadata))
}

// Build build a new CompositeMetadata.
func (c *CompositeMetadataBuilder) Build() (CompositeMetadata, error) {
	bf := bytes.Buffer{}
	for i := 0; i < len(c.k); i++ {
		switch mimeType := c.k[i].(type) {
		case MIME:
			bf.WriteByte(0x80 | byte(mimeType))
		case string:
			mimeTypeLen := len(mimeType)
			if mimeTypeLen > math.MaxInt8 {
				return nil, fmt.Errorf("length of MIME type is over %d", math.MaxInt8)
			}
			bf.WriteByte(byte(mimeTypeLen - 1))
			bf.Write([]byte(mimeType))
		default:
			panic("unreachable")
		}
		metadata := c.v[i]
		metadataLen := len(metadata)
		bf.Write(common.MustNewUint24(metadataLen).Bytes())
		if metadataLen > 0 {
			bf.Write(metadata)
		}
	}
	return bf.Bytes(), nil
}

// NewCompositeMetadataBuilder returns a CompositeMetadata builder.
func NewCompositeMetadataBuilder() *CompositeMetadataBuilder {
	return &CompositeMetadataBuilder{}
}

// NewCompositeMetadataBytes returns a CompositeMetadata.
func NewCompositeMetadataBytes(raw []byte) CompositeMetadata {
	return raw
}
