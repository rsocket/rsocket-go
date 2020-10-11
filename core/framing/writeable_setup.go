package framing

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// WriteableSetupFrame is writeable Setup frame.
type WriteableSetupFrame struct {
	writeableFrame
	version      core.Version
	keepalive    [4]byte
	lifetime     [4]byte
	token        []byte
	mimeMetadata []byte
	mimeData     []byte
	metadata     []byte
	data         []byte
}

// NewWriteableSetupFrame creates a new WriteableSetupFrame.
func NewWriteableSetupFrame(
	version core.Version,
	timeBetweenKeepalive,
	maxLifetime time.Duration,
	token []byte,
	mimeMetadata []byte,
	mimeData []byte,
	data []byte,
	metadata []byte,
	lease bool,
) *WriteableSetupFrame {
	var flag core.FrameFlag
	if l := len(token); l > 0 {
		flag |= core.FlagResume
	}
	if lease {
		flag |= core.FlagLease
	}
	if l := len(metadata); l > 0 {
		flag |= core.FlagMetadata
	}
	h := core.NewFrameHeader(0, core.FrameTypeSetup, flag)
	t := newWriteableFrame(h)

	var a, b [4]byte
	binary.BigEndian.PutUint32(a[:], uint32(common.ToMilliseconds(timeBetweenKeepalive)))
	binary.BigEndian.PutUint32(b[:], uint32(common.ToMilliseconds(maxLifetime)))
	return &WriteableSetupFrame{
		writeableFrame: t,
		version:        version,
		keepalive:      a,
		lifetime:       b,
		token:          token,
		mimeMetadata:   mimeMetadata,
		mimeData:       mimeData,
		metadata:       metadata,
		data:           data,
	}
}

// WriteTo writes frame to writer.
func (s WriteableSetupFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = s.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	wrote, err = s.version.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	var v int
	v, err = w.Write(s.keepalive[:])
	if err != nil {
		return
	}
	n += int64(v)

	v, err = w.Write(s.lifetime[:])
	if err != nil {
		return
	}
	n += int64(v)

	if s.header.Flag().Check(core.FlagResume) {
		tokenLen := len(s.token)
		err = binary.Write(w, binary.BigEndian, uint16(tokenLen))
		if err != nil {
			return
		}
		n += 2
		v, err = w.Write(s.token)
		if err != nil {
			return
		}
		n += int64(v)
	}

	lenMimeMetadata := len(s.mimeMetadata)
	v, err = w.Write([]byte{byte(lenMimeMetadata)})
	if err != nil {
		return
	}
	n += int64(v)
	v, err = w.Write(s.mimeMetadata)
	if err != nil {
		return
	}
	n += int64(v)

	lenMimeData := len(s.mimeData)
	v, err = w.Write([]byte{byte(lenMimeData)})
	if err != nil {
		return
	}
	n += int64(v)
	v, err = w.Write(s.mimeData)
	if err != nil {
		return
	}
	n += int64(v)

	wrote, err = writePayload(w, s.data, s.metadata)
	if err != nil {
		return
	}
	n += wrote
	return
}

// Len returns length of frame.
func (s WriteableSetupFrame) Len() int {
	n := _minSetupFrameLen + CalcPayloadFrameSize(s.data, s.metadata)
	n += len(s.mimeData) + len(s.mimeMetadata)
	if l := len(s.token); l > 0 {
		n += 2 + len(s.token)
	}
	return n
}
