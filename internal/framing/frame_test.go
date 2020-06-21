package framing_test

import (
	"bytes"
	"math"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	. "github.com/rsocket/rsocket-go/internal/framing"
	"github.com/stretchr/testify/assert"
)

const _sid uint32 = 1

func TestFrameCancel(t *testing.T) {
	f := NewCancelFrame(_sid)
	checkBasic(t, f, FrameTypeCancel)
	f2 := NewCancelFrameSupport(_sid)
	checkBytes(t, f, f2)
}

func TestFrameError(t *testing.T) {
	errData := []byte(common.RandAlphanumeric(100))
	f := NewErrorFrame(_sid, common.ErrorCodeApplicationError, errData)
	checkBasic(t, f, FrameTypeError)
	assert.Equal(t, common.ErrorCodeApplicationError, f.ErrorCode())
	assert.Equal(t, errData, f.ErrorData())
	assert.NotEmpty(t, f.Error())
	f2 := NewErrorFrame(_sid, common.ErrorCodeApplicationError, errData)
	checkBytes(t, f, f2)
}

func TestFrameFNF(t *testing.T) {
	b := []byte(common.RandAlphanumeric(100))
	// Without Metadata
	f := NewFireAndForgetFrame(_sid, b, nil, FlagNext)
	checkBasic(t, f, FrameTypeRequestFNF)
	assert.Equal(t, b, f.Data())
	metadata, ok := f.Metadata()
	assert.False(t, ok)
	assert.Nil(t, metadata)
	assert.True(t, f.Header().Flag().Check(FlagNext))
	assert.False(t, f.Header().Flag().Check(FlagMetadata))
	f2 := NewFireAndForgetFrameSupport(_sid, b, nil, FlagNext)
	checkBytes(t, f, f2)

	// With Metadata
	f = NewFireAndForgetFrame(_sid, nil, b, FlagNext)
	checkBasic(t, f, FrameTypeRequestFNF)
	assert.Empty(t, f.Data())
	metadata, ok = f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, metadata)
	assert.True(t, f.Header().Flag().Check(FlagNext))
	assert.True(t, f.Header().Flag().Check(FlagMetadata))
	f2 = NewFireAndForgetFrameSupport(_sid, nil, b, FlagNext)
	checkBytes(t, f, f2)
}

func TestFrameKeepalive(t *testing.T) {
	pos := uint64(common.RandIntn(math.MaxInt32))
	d := []byte(common.RandAlphanumeric(100))
	f := NewKeepaliveFrame(pos, d, true)
	checkBasic(t, f, FrameTypeKeepalive)
	assert.Equal(t, d, f.Data())
	assert.Equal(t, pos, f.LastReceivedPosition())
	assert.True(t, f.Header().Flag().Check(FlagRespond))
	f2 := NewKeepaliveFrameSupport(pos, d, true)
	checkBytes(t, f, f2)
}

func TestFrameLease(t *testing.T) {
	metadata := []byte("foobar")
	n := uint32(4444)
	f := NewLeaseFrame(time.Second, n, metadata)
	checkBasic(t, f, FrameTypeLease)
	assert.Equal(t, time.Second, f.TimeToLive())
	assert.Equal(t, n, f.NumberOfRequests())
	assert.Equal(t, metadata, f.Metadata())
	f2 := NewLeaseFrameSupport(time.Second, n, metadata)
	checkBytes(t, f, f2)
}

func TestFrameMetadataPush(t *testing.T) {
	metadata := []byte("foobar")
	f := NewMetadataPushFrame(metadata)
	checkBasic(t, f, FrameTypeMetadataPush)
	metadata2, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, metadata, metadata2)
	f2 := NewMetadataPushFrameSupport(metadata)
	checkBytes(t, f, f2)
}

func TestPayloadFrame(t *testing.T) {
	b := []byte("foobar")
	f := NewPayloadFrame(_sid, b, b, FlagNext)
	checkBasic(t, f, FrameTypePayload)
	m, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, f.Data())
	assert.Equal(t, b, m)
	assert.Equal(t, FlagNext|FlagMetadata, f.Header().Flag())
	f2 := NewPayloadFrameSupport(_sid, b, b, FlagNext)
	checkBytes(t, f, f2)
}

func TestFrameRequestChannel(t *testing.T) {
	b := []byte("foobar")
	n := uint32(1)
	f := NewRequestChannelFrame(_sid, n, b, b, FlagNext)
	checkBasic(t, f, FrameTypeRequestChannel)
	assert.Equal(t, n, f.InitialRequestN())
	assert.Equal(t, b, f.Data())
	m, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, m)
	f2 := NewRequestChannelFrameSupport(_sid, n, b, b, FlagNext)
	checkBytes(t, f, f2)
}

func TestFrameRequestN(t *testing.T) {
	n := uint32(1234)
	f := NewRequestNFrame(_sid, n, 0)
	checkBasic(t, f, FrameTypeRequestN)
	assert.Equal(t, n, f.N())
	f2 := NewRequestNFrameSupport(_sid, n, 0)
	checkBytes(t, f, f2)
}

func TestFrameRequestResponse(t *testing.T) {
	b := []byte("foobar")
	f := NewRequestResponseFrame(_sid, b, b, FlagNext)
	checkBasic(t, f, FrameTypeRequestResponse)
	assert.Equal(t, b, f.Data())
	m, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, m)
	assert.Equal(t, FlagNext|FlagMetadata, f.Header().Flag())
	f2 := NewRequestResponseFrameSupport(_sid, b, b, FlagNext)
	checkBytes(t, f, f2)
}

func TestFrameRequestStream(t *testing.T) {
	b := []byte("foobar")
	n := uint32(1234)
	f := NewRequestStreamFrame(_sid, n, b, b, FlagNext)
	checkBasic(t, f, FrameTypeRequestStream)
	assert.Equal(t, b, f.Data())
	assert.Equal(t, n, f.InitialRequestN())
	m, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, m)
	f2 := NewRequestStreamFrameSupport(_sid, n, b, b, FlagNext)
	checkBytes(t, f, f2)
}

func TestFrameResume(t *testing.T) {
	v := common.NewVersion(3, 1)
	token := []byte("hello")
	p1 := uint64(333)
	p2 := uint64(444)
	f := NewResumeFrame(v, token, p1, p2)
	checkBasic(t, f, FrameTypeResume)
	assert.Equal(t, token, f.Token())
	assert.Equal(t, p1, f.FirstAvailableClientPosition())
	assert.Equal(t, p2, f.LastReceivedServerPosition())
	assert.Equal(t, v.Major(), f.Version().Major())
	assert.Equal(t, v.Minor(), f.Version().Minor())
	f2 := NewResumeFrameSupport(v, token, p1, p2)
	checkBytes(t, f, f2)
}

func TestFrameResumeOK(t *testing.T) {
	pos := uint64(1234)
	f := NewResumeOKFrame(pos)
	checkBasic(t, f, FrameTypeResumeOK)
	assert.Equal(t, pos, f.LastReceivedClientPosition())
	f2 := NewResumeOKFrameSupport(pos)
	checkBytes(t, f, f2)
}

func TestFrameSetup(t *testing.T) {
	v := common.NewVersion(3, 1)
	timeKeepalive := 20 * time.Second
	maxLifetime := time.Minute + 30*time.Second
	var token []byte
	mimeData := []byte("application/binary")
	mimeMetadata := []byte("application/binary")
	d := []byte("你好")
	m := []byte("世界")
	f := NewSetupFrame(v, timeKeepalive, maxLifetime, token, mimeMetadata, mimeData, d, m, false)
	checkBasic(t, f, FrameTypeSetup)
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

	fs := NewSetupFrameSupport(v, timeKeepalive, maxLifetime, token, mimeMetadata, mimeData, d, m, false)

	checkBytes(t, f, fs)
}

func checkBasic(t *testing.T, f Frame, typ FrameType) {
	sid := _sid
	switch typ {
	case FrameTypeKeepalive, FrameTypeSetup, FrameTypeLease, FrameTypeResume, FrameTypeResumeOK, FrameTypeMetadataPush:
		sid = 0
	}
	assert.Equal(t, sid, f.Header().StreamID(), "wrong frame stream id")
	assert.NoError(t, f.Validate(), "validate frame type failed")
	assert.Equal(t, typ, f.Header().Type(), "frame type doesn't match")
	assert.True(t, f.Header().Type().String() != "UNKNOWN")
	go func() {
		f.Done()
	}()
	<-f.DoneNotify()
}

func checkBytes(t *testing.T, a Frame, b FrameSupport) {
	assert.Equal(t, a.Len(), b.Len())
	bf1, bf2 := &bytes.Buffer{}, &bytes.Buffer{}
	_, err := a.WriteTo(bf1)
	assert.NoError(t, err, "write failed")
	_, err = b.WriteTo(bf2)
	assert.NoError(t, err, "write failed")
	b1, b2 := bf1.Bytes(), bf2.Bytes()
	assert.Equal(t, b1, b2, "bytes doesn't match")
	bf := common.NewByteBuff()
	_, _ = bf.Write(b1[HeaderLen:])
	raw := NewRawFrame(ParseFrameHeader(b1[:HeaderLen]), bf)
	_, err = FromRawFrame(raw)
	assert.NoError(t, err, "create from raw failed")
}
