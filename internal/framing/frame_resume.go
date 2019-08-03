package framing

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/rsocket/rsocket-go/internal/common"
)

var errResumeTokenTooLarge = errors.New("max length of resume token is 65535")

// FrameResume represents a frame of Resume.
type FrameResume struct {
	*BaseFrame
}

func (p *FrameResume) String() string {
	return fmt.Sprintf(
		"FrameResume{%s,version=%s,token=0x%02x,lastReceivedServerPosition=%d,firstAvailableClientPosition=%d}",
		p.header, p.Version(), p.Token(), p.LastReceivedServerPosition(), p.FirstAvailableClientPosition(),
	)
}

// Validate validate current frame.
func (p *FrameResume) Validate() (err error) {
	return
}

// Version returns version.
func (p *FrameResume) Version() common.Version {
	raw := p.body.Bytes()
	major := binary.BigEndian.Uint16(raw)
	minor := binary.BigEndian.Uint16(raw[2:])
	return [2]uint16{major, minor}
}

// Token returns resume token in bytes.
func (p *FrameResume) Token() []byte {
	raw := p.body.Bytes()
	tokenLen := binary.BigEndian.Uint16(raw[4:6])
	return raw[6 : 6+tokenLen]
}

// LastReceivedServerPosition returns last received server position.
func (p *FrameResume) LastReceivedServerPosition() uint64 {
	raw := p.body.Bytes()
	offset := 6 + binary.BigEndian.Uint16(raw[4:6])
	return binary.BigEndian.Uint64(raw[offset:])
}

// FirstAvailableClientPosition returns first available client position.
func (p *FrameResume) FirstAvailableClientPosition() uint64 {
	raw := p.body.Bytes()
	offset := 6 + binary.BigEndian.Uint16(raw[4:6]) + 8
	return binary.BigEndian.Uint64(raw[offset:])
}

// NewFrameResume creates a new frame of Resume.
func NewFrameResume(version common.Version, token []byte, firstAvailableClientPosition, lastReceivedServerPosition uint64) *FrameResume {
	n := len(token)
	if n > math.MaxUint16 {
		panic(errResumeTokenTooLarge)
	}

	bf := common.New()
	_, err := bf.Write(version.Bytes())
	if err != nil {
		panic(err)
	}
	var b8 [8]byte
	binary.BigEndian.PutUint16(b8[:2], uint16(n))
	if _, err := bf.Write(b8[:2]); err != nil {
		panic(err)
	}
	if n > 0 {
		if _, err = bf.Write(token); err != nil {
			panic(err)
		}
	}
	binary.BigEndian.PutUint64(b8[:], lastReceivedServerPosition)
	if _, err = bf.Write(b8[:]); err != nil {
		panic(err)
	}
	binary.BigEndian.PutUint64(b8[:], firstAvailableClientPosition)
	if _, err = bf.Write(b8[:]); err != nil {
		panic(err)
	}
	return &FrameResume{
		NewBaseFrame(NewFrameHeader(0, FrameTypeResume), bf),
	}
}
