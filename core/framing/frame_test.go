package framing

import (
	"bytes"
	"math"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/stretchr/testify/assert"
)

const _sid uint32 = 1234

func TestFromBytes(t *testing.T) {
	// empty
	_, err := FromBytes([]byte{})
	assert.Error(t, err, "should be error")

	b := &bytes.Buffer{}
	frame := NewWriteableRequestResponseFrame(42, []byte("fake-data"), []byte("fake-metadata"), 0)
	_, _ = frame.WriteTo(b)
	frameActual, err := FromBytes(b.Bytes())
	assert.NoError(t, err, "should not be error")
	defer frameActual.Release()
	assert.Equal(t, frame.Header(), frameActual.Header(), "header does not match")
	assert.Equal(t, frame.Len(), frameActual.Len())
}

func TestFrameCancel(t *testing.T) {
	f := NewCancelFrame(_sid)
	defer f.Release()
	checkBasic(t, f, core.FrameTypeCancel)
	f2 := NewWriteableCancelFrame(_sid)
	checkBytes(t, f, f2)
}

func TestFrameError(t *testing.T) {
	errData := []byte(common.RandAlphanumeric(10))
	f := NewErrorFrame(_sid, core.ErrorCodeApplicationError, errData)
	defer f.Release()
	checkBasic(t, f, core.FrameTypeError)
	assert.Equal(t, core.ErrorCodeApplicationError, f.ErrorCode())
	assert.Equal(t, errData, f.ErrorData())
	assert.NotEmpty(t, f.Error())
	f2 := NewWriteableErrorFrame(_sid, core.ErrorCodeApplicationError, errData)
	checkBytes(t, f, f2)

	e, ok := f.ToError().(core.CustomError)
	assert.True(t, ok, "should implement CustomError interface")
	assert.Equal(t, core.ErrorCodeApplicationError, e.ErrorCode())
	assert.Equal(t, errData, e.ErrorData())
	assert.NotEmpty(t, e.Error())
}

func TestFrameFNF(t *testing.T) {
	b := []byte(common.RandAlphanumeric(100))
	// Without Metadata
	f := NewFireAndForgetFrame(_sid, b, nil, core.FlagNext)
	checkBasic(t, f, core.FrameTypeRequestFNF)
	assert.Equal(t, b, f.Data())
	_ = f.DataUTF8()
	_, _ = f.MetadataUTF8()
	metadata, ok := f.Metadata()
	assert.False(t, ok)
	assert.Nil(t, metadata)
	assert.True(t, f.Header().Flag().Check(core.FlagNext))
	assert.False(t, f.Header().Flag().Check(core.FlagMetadata))
	f2 := NewWriteableFireAndForgetFrame(_sid, b, nil, core.FlagNext)
	checkBytes(t, f, f2)
	f.Release()

	// With Metadata
	f = NewFireAndForgetFrame(_sid, nil, b, core.FlagNext)
	checkBasic(t, f, core.FrameTypeRequestFNF)
	assert.Empty(t, f.Data())
	metadata, ok = f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, metadata)
	assert.True(t, f.Header().Flag().Check(core.FlagNext))
	assert.True(t, f.Header().Flag().Check(core.FlagMetadata))
	f2 = NewWriteableFireAndForgetFrame(_sid, nil, b, core.FlagNext)
	checkBytes(t, f, f2)
	f.Release()
}

func TestFrameKeepalive(t *testing.T) {
	pos := uint64(common.RandIntn(math.MaxInt32))
	d := []byte(common.RandAlphanumeric(100))
	f := NewKeepaliveFrame(pos, d, true)
	defer f.Release()
	checkBasic(t, f, core.FrameTypeKeepalive)
	assert.Equal(t, d, f.Data())
	assert.Equal(t, pos, f.LastReceivedPosition())
	assert.True(t, f.Header().Flag().Check(core.FlagRespond))
	f2 := NewWriteableKeepaliveFrame(pos, d, true)
	checkBytes(t, f, f2)
}

func TestFrameLease(t *testing.T) {
	metadata := []byte("foobar")
	n := uint32(4444)
	f := NewLeaseFrame(time.Second, n, metadata)
	defer f.Release()
	checkBasic(t, f, core.FrameTypeLease)
	assert.Equal(t, time.Second, f.TimeToLive())
	assert.Equal(t, n, f.NumberOfRequests())
	assert.Equal(t, metadata, f.Metadata())
	f2 := NewWriteableLeaseFrame(time.Second, n, metadata)
	checkBytes(t, f, f2)
}

func TestFrameMetadataPush(t *testing.T) {
	metadata := []byte("foobar")
	f := NewMetadataPushFrame(metadata)
	defer f.Release()
	assert.Nil(t, f.Data(), "should not be nil")
	assert.Equal(t, "", f.DataUTF8(), "should be zero string")
	checkBasic(t, f, core.FrameTypeMetadataPush)
	metadata2, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, metadata, metadata2)
	_, _ = f.MetadataUTF8()

	f2 := NewWriteableMetadataPushFrame(metadata)
	checkBytes(t, f, f2)
}

func TestPayloadFrame(t *testing.T) {
	b := []byte("foobar")
	f := NewPayloadFrame(_sid, b, b, core.FlagNext)
	defer f.Release()
	checkBasic(t, f, core.FrameTypePayload)
	m, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, f.Data())
	assert.Equal(t, b, m)
	_ = f.DataUTF8()
	_, _ = f.MetadataUTF8()
	assert.Equal(t, core.FlagNext|core.FlagMetadata, f.Header().Flag())
	f2 := NewWriteablePayloadFrame(_sid, b, b, core.FlagNext)
	checkBytes(t, f, f2)

	assert.Equal(t, b, f2.Data())
	_ = f2.DataUTF8()
	m2, ok := f2.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, m2)
	_, _ = f2.MetadataUTF8()
}

func TestFrameRequestChannel(t *testing.T) {
	b := []byte("foobar")
	n := uint32(1)
	f := NewRequestChannelFrame(_sid, n, b, b, core.FlagNext)
	defer f.Release()
	checkBasic(t, f, core.FrameTypeRequestChannel)
	assert.Equal(t, n, f.InitialRequestN())
	assert.Equal(t, b, f.Data())
	m, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, m)

	_ = f.DataUTF8()
	_, _ = f.MetadataUTF8()

	f2 := NewWriteableRequestChannelFrame(_sid, n, b, b, core.FlagNext)
	checkBytes(t, f, f2)
}

func TestFrameRequestN(t *testing.T) {
	n := uint32(1234)
	f := NewRequestNFrame(_sid, n, 0)
	defer f.Release()
	checkBasic(t, f, core.FrameTypeRequestN)
	assert.Equal(t, n, f.N())
	f2 := NewWriteableRequestNFrame(_sid, n, 0)
	checkBytes(t, f, f2)
}

func TestFrameRequestResponse(t *testing.T) {
	b := []byte("foobar")
	f := NewRequestResponseFrame(_sid, b, b, core.FlagNext)
	defer f.Release()
	checkBasic(t, f, core.FrameTypeRequestResponse)
	assert.Equal(t, b, f.Data())
	m, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, m)
	assert.Equal(t, core.FlagNext|core.FlagMetadata, f.Header().Flag())
	_ = f.DataUTF8()
	_, _ = f.MetadataUTF8()
	f2 := NewWriteableRequestResponseFrame(_sid, b, b, core.FlagNext)
	checkBytes(t, f, f2)
}

func TestFrameRequestStream(t *testing.T) {
	b := []byte("foobar")
	n := uint32(1234)
	f := NewRequestStreamFrame(_sid, n, b, b, core.FlagNext)
	defer f.Release()
	checkBasic(t, f, core.FrameTypeRequestStream)
	assert.Equal(t, b, f.Data())
	assert.Equal(t, n, f.InitialRequestN())
	m, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, m)
	_, _ = f.MetadataUTF8()
	_ = f.DataUTF8()
	f2 := NewWriteableRequestStreamFrame(_sid, n, b, b, core.FlagNext)
	checkBytes(t, f, f2)
}

func TestFrameResume(t *testing.T) {
	v := core.NewVersion(3, 1)
	token := []byte("hello")
	p1 := uint64(333)
	p2 := uint64(444)
	f := NewResumeFrame(v, token, p1, p2)
	defer f.Release()
	checkBasic(t, f, core.FrameTypeResume)
	assert.Equal(t, token, f.Token())
	assert.Equal(t, p1, f.FirstAvailableClientPosition())
	assert.Equal(t, p2, f.LastReceivedServerPosition())
	assert.Equal(t, v.Major(), f.Version().Major())
	assert.Equal(t, v.Minor(), f.Version().Minor())
	f2 := NewWriteableResumeFrame(v, token, p1, p2)
	checkBytes(t, f, f2)
}

func TestFrameResumeOK(t *testing.T) {
	pos := uint64(1234)
	f := NewResumeOKFrame(pos)
	defer f.Release()
	checkBasic(t, f, core.FrameTypeResumeOK)
	assert.Equal(t, pos, f.LastReceivedClientPosition())
	f2 := NewWriteableResumeOKFrame(pos)
	checkBytes(t, f, f2)
}

func TestFrameSetup(t *testing.T) {
	v := core.NewVersion(3, 1)
	timeKeepalive := 20 * time.Second
	maxLifetime := time.Minute + 30*time.Second
	var token []byte
	mimeData := []byte("application/binary")
	mimeMetadata := []byte("application/binary")
	d := []byte("你好")
	m := []byte("世界")
	f := NewSetupFrame(v, timeKeepalive, maxLifetime, token, mimeMetadata, mimeData, d, m, false)
	defer f.Release()
	checkBasic(t, f, core.FrameTypeSetup)
	assert.Equal(t, v.Major(), f.Version().Major())
	assert.Equal(t, v.Minor(), f.Version().Minor())
	assert.Equal(t, timeKeepalive, f.TimeBetweenKeepalive())
	assert.Equal(t, maxLifetime, f.MaxLifetime())
	assert.Equal(t, token, f.Token())
	assert.Equal(t, string(mimeData), f.DataMimeType())
	assert.Equal(t, string(mimeMetadata), f.MetadataMimeType())
	assert.Equal(t, d, f.Data())
	m2, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, m, m2)

	_ = f.DataUTF8()
	_, _ = f.MetadataUTF8()

	fs := NewWriteableSetupFrame(v, timeKeepalive, maxLifetime, token, mimeMetadata, mimeData, d, m, false)

	checkBytes(t, f, fs)
}

func checkBasic(t *testing.T, f core.BufferedFrame, typ core.FrameType) {
	sid := _sid
	switch typ {
	case core.FrameTypeKeepalive, core.FrameTypeSetup, core.FrameTypeLease, core.FrameTypeResume, core.FrameTypeResumeOK, core.FrameTypeMetadataPush:
		sid = 0
	}
	assert.Equal(t, sid, f.Header().StreamID(), "wrong frame stream id")
	assert.NoError(t, f.Validate(), "validate frame type failed")
	assert.Equal(t, typ, f.Header().Type(), "frame type doesn't match")
	assert.True(t, f.Header().Type().String() != "UNKNOWN")
}

func checkBytes(t *testing.T, a core.BufferedFrame, b core.WriteableFrame) {
	assert.NoError(t, a.Validate())
	assert.Equal(t, a.Len(), b.Len())
	bf1, bf2 := &bytes.Buffer{}, &bytes.Buffer{}
	_, err := a.WriteTo(bf1)
	assert.NoError(t, err, "write failed")
	_, err = b.WriteTo(bf2)
	assert.NoError(t, err, "write failed")
	b1, b2 := bf1.Bytes(), bf2.Bytes()
	assert.Equal(t, b1, b2, "bytes doesn't match")
	bf := common.BorrowByteBuff()
	defer common.ReturnByteBuff(bf)
	_, _ = bf.Write(b1)
	raw := newBufferedFrame(bf)
	_, err = convert(raw)
	assert.NoError(t, err, "create from raw failed")
}
