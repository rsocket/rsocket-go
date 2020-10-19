package framing

import (
	"io"
	"sync"

	"github.com/rsocket/rsocket-go/core"
)

var _writeablePayloadFramePool = sync.Pool{
	New: func() interface{} {
		return new(WriteablePayloadFrame)
	},
}

// WriteablePayloadFrame is writeable Payload frame.
type WriteablePayloadFrame struct {
	writeableFrame
	metadata []byte
	data     []byte
}

// NewWriteablePayloadFrame returns a new WriteablePayloadFrame.
func NewWriteablePayloadFrame(id uint32, data, metadata []byte, flag core.FrameFlag) *WriteablePayloadFrame {
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	result := _writeablePayloadFramePool.Get().(*WriteablePayloadFrame)
	core.ResetFrameHeader(result.header[:], id, core.FrameTypePayload, flag)
	result.data = data
	result.metadata = metadata
	result.doneHandler = nil
	return result
}

// Data returns data bytes.
func (p WriteablePayloadFrame) Data() []byte {
	return p.data
}

// Metadata returns metadata bytes.
func (p WriteablePayloadFrame) Metadata() (metadata []byte, ok bool) {
	ok = p.header.Flag().Check(core.FlagMetadata)
	if ok {
		metadata = p.metadata
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (p WriteablePayloadFrame) DataUTF8() (data string) {
	if p.data != nil {
		data = string(p.data)
	}
	return
}

// MetadataUTF8 returns metadata as UTF8 string.
func (p WriteablePayloadFrame) MetadataUTF8() (metadata string, ok bool) {
	ok = p.header.Flag().Check(core.FlagMetadata)
	if ok {
		metadata = string(p.metadata)
	}
	return
}

// WriteTo writes frame to writer.
func (p WriteablePayloadFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = p.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote
	wrote, err = writePayload(w, p.data, p.metadata)
	if err == nil {
		n += wrote
	}
	return
}

// Len returns length of frame.
func (p WriteablePayloadFrame) Len() int {
	return CalcPayloadFrameSize(p.data, p.metadata)
}

func (p *WriteablePayloadFrame) Done() {
	p.writeableFrame.Done()
	p.data = nil
	p.metadata = nil
	_writeablePayloadFramePool.Put(p)
}
