package framing_test

import (
	"encoding/hex"
	"log"
	"math"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	. "github.com/rsocket/rsocket-go/internal/framing"
	"github.com/stretchr/testify/assert"
)

const _sid uint32 = 1

func TestFrameCancel(t *testing.T) {
	f := NewFrameCancel(_sid)
	basicCheck(t, f, FrameTypeCancel)
}

func TestFrameError(t *testing.T) {
	errData := []byte(common.RandAlphanumeric(100))
	f := NewFrameError(_sid, common.ErrorCodeApplicationError, errData)
	basicCheck(t, f, FrameTypeError)
	assert.Equal(t, common.ErrorCodeApplicationError, f.ErrorCode())
	assert.Equal(t, errData, f.ErrorData())
	assert.NotEmpty(t, f.Error())
}

func TestFrameFNF(t *testing.T) {
	b := []byte(common.RandAlphanumeric(100))
	// Without Metadata
	f := NewFrameFNF(_sid, b, nil, FlagNext)
	basicCheck(t, f, FrameTypeRequestFNF)
	assert.Equal(t, b, f.Data())
	metadata, ok := f.Metadata()
	assert.False(t, ok)
	assert.Nil(t, metadata)
	assert.True(t, f.Header().Flag().Check(FlagNext))
	assert.False(t, f.Header().Flag().Check(FlagMetadata))
	// With Metadata
	f = NewFrameFNF(_sid, nil, b, FlagNext)
	basicCheck(t, f, FrameTypeRequestFNF)
	assert.Empty(t, f.Data())
	metadata, ok = f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, metadata)
	assert.True(t, f.Header().Flag().Check(FlagNext))
	assert.True(t, f.Header().Flag().Check(FlagMetadata))
}

func TestFrameKeepalive(t *testing.T) {
	pos := uint64(common.RandIntn(math.MaxInt32))
	d := []byte(common.RandAlphanumeric(100))
	f := NewFrameKeepalive(pos, d, true)
	basicCheck(t, f, FrameTypeKeepalive)
	assert.Equal(t, d, f.Data())
	assert.Equal(t, pos, f.LastReceivedPosition())
	assert.True(t, f.Header().Flag().Check(FlagRespond))
}

func TestFrameLease(t *testing.T) {
	metadata := []byte("foobar")
	n := uint32(4444)
	f := NewFrameLease(time.Second, n, metadata)
	basicCheck(t, f, FrameTypeLease)
	assert.Equal(t, time.Second, f.TimeToLive())
	assert.Equal(t, n, f.NumberOfRequests())
	assert.Equal(t, metadata, f.Metadata())
}

func TestFrameMetadataPush(t *testing.T) {
	metadata := []byte("foobar")
	f := NewFrameMetadataPush(metadata)
	basicCheck(t, f, FrameTypeMetadataPush)
	metadata2, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, metadata, metadata2)
}

func TestFramePayload(t *testing.T) {
	b := []byte("foobar")
	f := NewFramePayload(_sid, b, b, FlagNext)
	basicCheck(t, f, FrameTypePayload)
	m, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, f.Data())
	assert.Equal(t, b, m)
	assert.Equal(t, FlagNext|FlagMetadata, f.Header().Flag())
}

func TestFrameRequestChannel(t *testing.T) {
	b := []byte("foobar")
	n := uint32(1)
	f := NewFrameRequestChannel(_sid, n, b, b, FlagNext)
	basicCheck(t, f, FrameTypeRequestChannel)
	assert.Equal(t, n, f.InitialRequestN())
	assert.Equal(t, b, f.Data())
	m, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, m)
}

func TestFrameRequestN(t *testing.T) {
	n := uint32(1234)
	f := NewFrameRequestN(_sid, n)
	basicCheck(t, f, FrameTypeRequestN)
	assert.Equal(t, n, f.N())
}

func TestFrameRequestResponse(t *testing.T) {
	b := []byte("foobar")
	f := NewFrameRequestResponse(_sid, b, b, FlagNext)
	basicCheck(t, f, FrameTypeRequestResponse)
	assert.Equal(t, b, f.Data())
	m, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, m)
	assert.Equal(t, FlagNext|FlagMetadata, f.Header().Flag())
}

func TestFrameRequestStream(t *testing.T) {
	b := []byte("foobar")
	n := uint32(1234)
	f := NewFrameRequestStream(_sid, n, b, b, FlagNext)
	basicCheck(t, f, FrameTypeRequestStream)
	assert.Equal(t, b, f.Data())
	assert.Equal(t, n, f.InitialRequestN())
	m, ok := f.Metadata()
	assert.True(t, ok)
	assert.Equal(t, b, m)
}

func TestFrameResume(t *testing.T) {
	v := common.NewVersion(3, 1)
	token := []byte("hello")
	p1 := uint64(333)
	p2 := uint64(444)
	f := NewFrameResume(v, token, p1, p2)
	basicCheck(t, f, FrameTypeResume)
	assert.Equal(t, token, f.Token())
	assert.Equal(t, p1, f.FirstAvailableClientPosition())
	assert.Equal(t, p2, f.LastReceivedServerPosition())
	assert.Equal(t, v.Major(), f.Version().Major())
	assert.Equal(t, v.Minor(), f.Version().Minor())
}

func TestFrameResumeOK(t *testing.T) {
	pos := uint64(1234)
	f := NewResumeOK(pos)
	basicCheck(t, f, FrameTypeResumeOK)
	assert.Equal(t, pos, f.LastReceivedClientPosition())
}

func TestFrameSetup(t *testing.T) {
	v := common.NewVersion(3, 1)
	timeKeepalive := 30 * time.Second
	maxLifetime := 3 * timeKeepalive
	token := []byte("hello")
	mimeData := []byte("application/json")
	mimeMetadata := []byte("text/plain")
	d := []byte(`{"hello":"world"}`)
	m := []byte("foobar")
	f := NewFrameSetup(v, timeKeepalive, maxLifetime, token, mimeMetadata, mimeData, d, m, false)
	basicCheck(t, f, FrameTypeSetup)
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
}

func TestDecode_Payload(t *testing.T) {
	//s := "000000012940000005776f726c6468656c6c6f" // go
	//s := "00000001296000000966726f6d5f6a617661706f6e67" //java

	var all []string
	all = append(all, "0000000004400001000000004e2000015f90126170706c69636174696f6e2f62696e617279126170706c69636174696f6e2f62696e617279")
	all = append(all, "00000000090000000bb800000005")
	all = append(all, "00000000090000001b5800000005")
	all = append(all, "000000011100000000436c69656e74207265717565737420547565204f63742032322032303a31373a3333204353542032303139")
	all = append(all, "00000001286053657276657220526573706f6e736520547565204f63742032322032303a31373a3333204353542032303139")

	for _, s := range all {
		bs, err := hex.DecodeString(s)
		assert.NoError(t, err, "bad bytes")
		h := ParseFrameHeader(bs[:HeaderLen])
		//log.Println(h)
		bf := common.NewByteBuff()
		_, _ = bf.Write(bs[HeaderLen:])
		f, err := NewFromBase(NewBaseFrame(h, bf))
		assert.NoError(t, err, "decode failed")
		log.Println(f)
	}

	lease := NewFrameLease(3*time.Second, 5, nil)
	log.Println("actual:", hex.EncodeToString(lease.Bytes()))
	log.Println("should: 00000000090000000bb800000005")
}

func basicCheck(t *testing.T, f Frame, typ FrameType) {
	sid := _sid
	switch typ {
	case FrameTypeKeepalive, FrameTypeSetup, FrameTypeLease, FrameTypeResume, FrameTypeResumeOK, FrameTypeMetadataPush:
		sid = 0
	}
	assert.Equal(t, sid, f.Header().StreamID(), "wrong frame stream id")
	assert.NoError(t, f.Validate(), "validate frame type failed")
	assert.Equal(t, typ, f.Header().Type(), "frame type doesn't match")
	assert.NotEmpty(t, f.String(), "empty frame string")
}
