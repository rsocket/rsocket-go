package framing

import (
	"io"
	"sync"

	"github.com/rsocket/rsocket-go/core"
)

var _writeableRequestResponseFramePool = sync.Pool{
	New: func() interface{} {
		return new(WriteableRequestResponseFrame)
	},
}

// WriteableRequestResponseFrame is writeable RequestResponse frame.
type WriteableRequestResponseFrame struct {
	writeableFrame
	metadata []byte
	data     []byte
}

// NewWriteableRequestResponseFrame returns a new WriteableRequestResponseFrame.
func NewWriteableRequestResponseFrame(id uint32, data, metadata []byte, fg core.FrameFlag) *WriteableRequestResponseFrame {
	if len(metadata) > 0 {
		fg |= core.FlagMetadata
	}
	result := _writeableRequestResponseFramePool.Get().(*WriteableRequestResponseFrame)
	core.ResetFrameHeader(result.header[:], id, core.FrameTypeRequestResponse, fg)
	result.data = data
	result.metadata = metadata
	result.doneHandler = nil
	return result
}

// WriteTo writes frame to writer.
func (r WriteableRequestResponseFrame) WriteTo(w io.Writer) (n int64, err error) {
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

// Len returns length of frame.
func (r WriteableRequestResponseFrame) Len() int {
	return CalcPayloadFrameSize(r.data, r.metadata)
}

func (r *WriteableRequestResponseFrame) Done() {
	r.writeableFrame.Done()
	r.data = nil
	r.metadata = nil
	_writeableRequestResponseFramePool.Put(r)
}
