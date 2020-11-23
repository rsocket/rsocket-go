package framing

import (
	"encoding/binary"
	"io"
	"sync/atomic"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/u24"
)

// bufferedFrame is basic frame implementation.
type bufferedFrame struct {
	inner *common.ByteBuff
	refs  int32
}

func newBufferedFrame(inner *common.ByteBuff) *bufferedFrame {
	return &bufferedFrame{
		inner: inner,
		refs:  1,
	}
}

func (f *bufferedFrame) IncRef() int32 {
	return atomic.AddInt32(&f.refs, 1)
}

func (f *bufferedFrame) RefCnt() int32 {
	return atomic.LoadInt32(&f.refs)
}

func (f *bufferedFrame) Header() core.FrameHeader {
	if f.inner == nil {
		panic("frame has been released!")
	}
	b := f.inner.Bytes()
	_ = b[core.FrameHeaderLen-1]
	var h core.FrameHeader
	copy(h[:], b)
	return h
}

func (f *bufferedFrame) HasFlag(flag core.FrameFlag) bool {
	if f.inner == nil {
		panic("frame has been released!")
	}
	n := binary.BigEndian.Uint16(f.inner.Bytes()[4:6])
	return core.FrameFlag(n&0x03FF)&flag == flag
}

func (f *bufferedFrame) StreamID() uint32 {
	if f.inner == nil {
		panic("frame has been released!")
	}
	return binary.BigEndian.Uint32(f.inner.Bytes()[:4])
}

// Release releases resource.
func (f *bufferedFrame) Release() {
	if f != nil && f.inner != nil && atomic.AddInt32(&f.refs, -1) == 0 {
		common.ReturnByteBuff(f.inner)
		f.inner = nil
	}
}

// Body returns frame body.
func (f *bufferedFrame) Body() []byte {
	if f.inner == nil {
		return nil
	}
	b := f.inner.Bytes()
	_ = b[core.FrameHeaderLen-1]
	return b[core.FrameHeaderLen:]
}

// Len returns length of frame.
func (f *bufferedFrame) Len() int {
	if f.inner == nil {
		return 0
	}
	return f.inner.Len()
}

// WriteTo write frame to writer.
func (f *bufferedFrame) WriteTo(w io.Writer) (n int64, err error) {
	if f == nil || f.inner == nil {
		return
	}
	n, err = f.inner.WriteTo(w)
	return
}

func (f *bufferedFrame) trySeekMetadataLen(offset int) (n int, hasMetadata bool) {
	raw := f.Body()
	if offset > 0 {
		raw = raw[offset:]
	}
	hasMetadata = f.HasFlag(core.FlagMetadata)
	if !hasMetadata {
		return
	}
	if len(raw) < 3 {
		n = -1
	} else {
		n = u24.NewUint24Bytes(raw).AsInt()
	}
	return
}

func (f *bufferedFrame) trySliceMetadata(offset int) ([]byte, bool) {
	n, ok := f.trySeekMetadataLen(offset)
	if !ok || n < 0 {
		return nil, false
	}
	return f.Body()[offset+3 : offset+3+n], true
}

func (f *bufferedFrame) trySliceData(offset int) []byte {
	n, ok := f.trySeekMetadataLen(offset)
	if !ok {
		return f.Body()[offset:]
	}
	if n < 0 {
		return nil
	}
	return f.Body()[offset+n+3:]
}

func (f *bufferedFrame) bodyLen() int {
	if l := f.Len(); l > core.FrameHeaderLen {
		return f.Len() - core.FrameHeaderLen
	}
	return 0
}
