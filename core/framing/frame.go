package framing

import (
	"errors"
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

var errIncompleteFrame = errors.New("incomplete frame")

type baseWriteableFrame struct {
	header core.FrameHeader
	done   chan struct{}
}

// FromBytes creates frame from a byte slice.
func FromBytes(b []byte) (f core.BufferedFrame, err error) {
	if len(b) < core.FrameHeaderLen {
		err = errIncompleteFrame
		return
	}
	header := core.ParseFrameHeader(b[:core.FrameHeaderLen])
	bb := common.BorrowByteBuff()
	_, err = bb.Write(b[core.FrameHeaderLen:])
	if err != nil {
		common.ReturnByteBuff(bb)
		return
	}
	raw := newBaseDefaultFrame(header, bb)
	f, err = FromRawFrame(raw)
	if err != nil {
		common.ReturnByteBuff(bb)
	}
	return
}

func (t baseWriteableFrame) Header() core.FrameHeader {
	return t.header
}

// Done can be invoked when a frame has been been processed.
func (t baseWriteableFrame) Done() (closed bool) {
	defer func() {
		closed = recover() != nil
	}()
	close(t.done)
	return
}

// DoneNotify notify when frame has been done.
func (t baseWriteableFrame) DoneNotify() <-chan struct{} {
	return t.done
}

// baseDefaultFrame is basic frame implementation.
type baseDefaultFrame struct {
	header core.FrameHeader
	body   *common.ByteBuff
}

func (f *baseDefaultFrame) Header() core.FrameHeader {
	return f.header
}

// Release releases resource.
func (f *baseDefaultFrame) Release() {
	if f != nil && f.body != nil {
		common.ReturnByteBuff(f.body)
		f.body = nil
	}
}

// Body returns frame body.
func (f *baseDefaultFrame) Body() *common.ByteBuff {
	return f.body
}

// Len returns length of frame.
func (f *baseDefaultFrame) Len() int {
	if f.body == nil {
		return core.FrameHeaderLen
	}
	return core.FrameHeaderLen + f.body.Len()
}

// WriteTo write frame to writer.
func (f *baseDefaultFrame) WriteTo(w io.Writer) (n int64, err error) {
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

func (f *baseDefaultFrame) trySeekMetadataLen(offset int) (n int, hasMetadata bool) {
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

func (f *baseDefaultFrame) trySliceMetadata(offset int) ([]byte, bool) {
	n, ok := f.trySeekMetadataLen(offset)
	if !ok || n < 0 {
		return nil, false
	}
	return f.body.Bytes()[offset+3 : offset+3+n], true
}

func (f *baseDefaultFrame) trySliceData(offset int) []byte {
	n, ok := f.trySeekMetadataLen(offset)
	if !ok {
		return f.body.Bytes()[offset:]
	}
	if n < 0 {
		return nil
	}
	return f.body.Bytes()[offset+n+3:]
}

func newBaseWriteableFrame(header core.FrameHeader) baseWriteableFrame {
	return baseWriteableFrame{
		header: header,
		done:   make(chan struct{}),
	}
}

func newBaseDefaultFrame(header core.FrameHeader, body *common.ByteBuff) *baseDefaultFrame {
	return &baseDefaultFrame{
		header: header,
		body:   body,
	}
}
