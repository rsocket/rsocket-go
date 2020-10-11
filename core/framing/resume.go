package framing

import (
	"encoding/binary"
	"errors"
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
	*bufferedFrame
}

// NewResumeFrame creates a new ResumeFrame.
func NewResumeFrame(version core.Version, token []byte, firstAvailableClientPosition, lastReceivedServerPosition uint64) *ResumeFrame {
	n := len(token)
	if n > math.MaxUint16 {
		panic(errResumeTokenTooLarge)
	}

	bb := common.BorrowByteBuff()

	if err := core.WriteFrameHeader(bb, 0, core.FrameTypeResume, 0); err != nil {
		common.ReturnByteBuff(bb)
		panic(err)
	}

	if _, err := bb.Write(version.Bytes()); err != nil {
		common.ReturnByteBuff(bb)
		panic(err)
	}
	if err := binary.Write(bb, binary.BigEndian, uint16(n)); err != nil {
		common.ReturnByteBuff(bb)
		panic(err)
	}
	if n > 0 {
		if _, err := bb.Write(token); err != nil {
			common.ReturnByteBuff(bb)
			panic(err)
		}
	}
	if err := binary.Write(bb, binary.BigEndian, lastReceivedServerPosition); err != nil {
		common.ReturnByteBuff(bb)
		panic(err)
	}
	if err := binary.Write(bb, binary.BigEndian, firstAvailableClientPosition); err != nil {
		common.ReturnByteBuff(bb)
		panic(err)
	}

	return &ResumeFrame{
		bufferedFrame: newBufferedFrame(bb),
	}
}

// Validate validate current frame.
func (r *ResumeFrame) Validate() (err error) {
	if r.bodyLen() < _minResumeLength {
		err = errIncompleteFrame
	}
	return
}

// Version returns version.
func (r *ResumeFrame) Version() core.Version {
	raw := r.Body()
	major := binary.BigEndian.Uint16(raw)
	minor := binary.BigEndian.Uint16(raw[2:])
	return [2]uint16{major, minor}
}

// Token returns resume token in bytes.
func (r *ResumeFrame) Token() []byte {
	raw := r.Body()
	tokenLen := binary.BigEndian.Uint16(raw[4:6])
	return raw[6 : 6+tokenLen]
}

// LastReceivedServerPosition returns last received server position.
func (r *ResumeFrame) LastReceivedServerPosition() uint64 {
	raw := r.Body()
	offset := 6 + binary.BigEndian.Uint16(raw[4:6])
	return binary.BigEndian.Uint64(raw[offset:])
}

// FirstAvailableClientPosition returns first available client position.
func (r *ResumeFrame) FirstAvailableClientPosition() uint64 {
	raw := r.Body()
	offset := 6 + binary.BigEndian.Uint16(raw[4:6]) + 8
	return binary.BigEndian.Uint64(raw[offset:])
}
