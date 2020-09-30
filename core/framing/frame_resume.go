package framing

import (
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

var errResumeTokenTooLarge = errors.New("max length of resume token is 65535")

const (
	_lenVersion      = 4
	_lenTokenLength  = 2
	_lenLastRecvPos  = 8
	_lenFirstPos     = 8
	_minResumeLength = _lenVersion + _lenTokenLength + _lenLastRecvPos + _lenFirstPos
)

// ResumeFrame is Resume frame.
type ResumeFrame struct {
	*baseDefaultFrame
}

// WriteableResumeFrame is writeable Resume frame.
type WriteableResumeFrame struct {
	baseWriteableFrame
	version  core.Version
	token    []byte
	posFirst [8]byte
	posLast  [8]byte
}

// Validate validate current frame.
func (r *ResumeFrame) Validate() (err error) {
	if r.body.Len() < _minResumeLength {
		err = errIncompleteFrame
	}
	return
}

// Version returns version.
func (r *ResumeFrame) Version() core.Version {
	raw := r.body.Bytes()
	major := binary.BigEndian.Uint16(raw)
	minor := binary.BigEndian.Uint16(raw[2:])
	return [2]uint16{major, minor}
}

// Token returns resume token in bytes.
func (r *ResumeFrame) Token() []byte {
	raw := r.body.Bytes()
	tokenLen := binary.BigEndian.Uint16(raw[4:6])
	return raw[6 : 6+tokenLen]
}

// LastReceivedServerPosition returns last received server position.
func (r *ResumeFrame) LastReceivedServerPosition() uint64 {
	raw := r.body.Bytes()
	offset := 6 + binary.BigEndian.Uint16(raw[4:6])
	return binary.BigEndian.Uint64(raw[offset:])
}

// FirstAvailableClientPosition returns first available client position.
func (r *ResumeFrame) FirstAvailableClientPosition() uint64 {
	raw := r.body.Bytes()
	offset := 6 + binary.BigEndian.Uint16(raw[4:6]) + 8
	return binary.BigEndian.Uint64(raw[offset:])
}

// WriteTo writes frame to writer.
func (r WriteableResumeFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = r.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	var v int

	v, err = w.Write(r.version.Bytes())
	if err != nil {
		return
	}
	n += int64(v)

	lenToken := uint16(len(r.token))
	err = binary.Write(w, binary.BigEndian, lenToken)
	if err != nil {
		return
	}
	n += 2

	v, err = w.Write(r.token)
	if err != nil {
		return
	}
	n += int64(v)

	v, err = w.Write(r.posLast[:])
	if err != nil {
		return
	}
	n += int64(v)

	v, err = w.Write(r.posFirst[:])
	if err != nil {
		return
	}
	n += int64(v)

	return
}

// Len returns length of frame.
func (r WriteableResumeFrame) Len() int {
	return core.FrameHeaderLen + _lenTokenLength + _lenFirstPos + _lenLastRecvPos + _lenVersion + len(r.token)
}

// NewWriteableResumeFrame creates a new WriteableResumeFrame.
func NewWriteableResumeFrame(version core.Version, token []byte, firstAvailableClientPosition, lastReceivedServerPosition uint64) *WriteableResumeFrame {
	h := core.NewFrameHeader(0, core.FrameTypeResume, 0)
	t := newBaseWriteableFrame(h)
	var a, b [8]byte
	binary.BigEndian.PutUint64(a[:], firstAvailableClientPosition)
	binary.BigEndian.PutUint64(b[:], lastReceivedServerPosition)

	return &WriteableResumeFrame{
		baseWriteableFrame: t,
		version:            version,
		token:              token,
		posFirst:           a,
		posLast:            b,
	}
}

// NewResumeFrame creates a new ResumeFrame.
func NewResumeFrame(version core.Version, token []byte, firstAvailableClientPosition, lastReceivedServerPosition uint64) *ResumeFrame {
	n := len(token)
	if n > math.MaxUint16 {
		panic(errResumeTokenTooLarge)
	}
	b := common.BorrowByteBuff()
	if _, err := b.Write(version.Bytes()); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	if err := binary.Write(b, binary.BigEndian, uint16(n)); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	if n > 0 {
		if _, err := b.Write(token); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
	}
	if err := binary.Write(b, binary.BigEndian, lastReceivedServerPosition); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	if err := binary.Write(b, binary.BigEndian, firstAvailableClientPosition); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	return &ResumeFrame{
		newBaseDefaultFrame(core.NewFrameHeader(0, core.FrameTypeResume, 0), b),
	}
}
