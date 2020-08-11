package framing

import (
	"errors"
	"fmt"
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

var errIncompleteFrame = errors.New("incomplete frame")

type tinyFrame struct {
	header core.FrameHeader
	done   chan struct{}
}

func (t *tinyFrame) Header() core.FrameHeader {
	return t.header
}

// Done can be invoked when a frame has been been processed.
func (t *tinyFrame) Done() (closed bool) {
	defer func() {
		if e := recover(); e != nil {
			closed = true
		}
	}()
	close(t.done)
	return
}

// DoneNotify notify when frame has been done.
func (t *tinyFrame) DoneNotify() <-chan struct{} {
	return t.done
}

// RawFrame is basic frame implementation.
type RawFrame struct {
	*tinyFrame
	body *common.ByteBuff
}

// Body returns frame body.
func (f *RawFrame) Body() *common.ByteBuff {
	return f.body
}

// Len returns length of frame.
func (f *RawFrame) Len() int {
	if f.body == nil {
		return core.FrameHeaderLen
	}
	return core.FrameHeaderLen + f.body.Len()
}

// WriteTo write frame to writer.
func (f *RawFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = f.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote
	if f.body != nil {
		wrote, err = f.body.WriteTo(w)
		if err != nil {
			return
		}
		n += wrote
	}
	return
}

func (f *RawFrame) trySeekMetadataLen(offset int) (n int, hasMetadata bool) {
	raw := f.body.Bytes()
	if offset > 0 {
		raw = raw[offset:]
	}
	hasMetadata = f.header.Flag().Check(core.FlagMetadata)
	if !hasMetadata {
		return
	}
	if len(raw) < 3 {
		n = -1
	} else {
		n = common.NewUint24Bytes(raw).AsInt()
	}
	return
}

func (f *RawFrame) trySliceMetadata(offset int) ([]byte, bool) {
	n, ok := f.trySeekMetadataLen(offset)
	if !ok || n < 0 {
		return nil, false
	}
	return f.body.Bytes()[offset+3 : offset+3+n], true
}

func (f *RawFrame) trySliceData(offset int) []byte {
	n, ok := f.trySeekMetadataLen(offset)
	if !ok {
		return f.body.Bytes()[offset:]
	}
	if n < 0 {
		return nil
	}
	return f.body.Bytes()[offset+n+3:]
}

func newTinyFrame(header core.FrameHeader) *tinyFrame {
	return &tinyFrame{
		header: header,
		done:   make(chan struct{}),
	}
}

// NewRawFrame returns a new RawFrame.
func NewRawFrame(header core.FrameHeader, body *common.ByteBuff) *RawFrame {
	return &RawFrame{
		tinyFrame: newTinyFrame(header),
		body:      body,
	}
}

// FromBytes creates frame from a byte slice.
func FromBytes(b []byte) (core.Frame, error) {
	if len(b) < core.FrameHeaderLen {
		return nil, errIncompleteFrame
	}
	header := core.ParseFrameHeader(b[:core.FrameHeaderLen])
	bb := common.NewByteBuff()
	_, err := bb.Write(b[core.FrameHeaderLen:])
	if err != nil {
		return nil, err
	}
	raw := NewRawFrame(header, bb)
	return FromRawFrame(raw)
}

func PrintFrame(f core.WriteableFrame) string {
	// TODO: print frame
	return fmt.Sprintf("%+v", f)
}
