package transport_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestLengthBasedFrameDecoder_ReadBroken(t *testing.T) {
	b := &bytes.Buffer{}

	_, _ = common.MustNewUint24(5).WriteTo(b)
	_, _ = b.Write([]byte{'_', 'f', 'a', 'k', 'e'})
	decoder := transport.NewLengthBasedFrameDecoder(b)
	_, err := decoder.Read()
	assert.Equal(t, transport.ErrIncompleteHeader, err, "should be incomplete header error")

	b.Reset()
	_, _ = b.Write([]byte{0, 0, 0, 'f', 'a', 'k', 'e'})
	decoder = transport.NewLengthBasedFrameDecoder(b)
	_, err = decoder.Read()
	assert.Equal(t, core.ErrInvalidFrameLength, err, "should be invalid length error")

	b.Reset()
	b.Write([]byte{0, 0})
	decoder = transport.NewLengthBasedFrameDecoder(b)
	_, err = decoder.Read()
	assert.Equal(t, io.EOF, err, "should read nothing")

	b.Reset()
	_, _ = common.MustNewUint24(10).WriteTo(b)
	var notEnough [7]byte
	_, _ = b.Write(notEnough[:])
	decoder = transport.NewLengthBasedFrameDecoder(b)
	_, err = decoder.Read()
	assert.Equal(t, io.EOF, err, "should read nothing")
}

func TestLengthBasedFrameDecoder_Read(t *testing.T) {
	b := &bytes.Buffer{}
	frames := []core.BufferedFrame{
		framing.NewSetupFrame(
			core.DefaultVersion,
			30*time.Second,
			90*time.Second,
			nil,
			[]byte("text/plain"),
			[]byte("text/plain"),
			[]byte("fake-data"),
			[]byte("fake-metadata"),
			false,
		),
		framing.NewKeepaliveFrame(0, []byte("fake-data"), true),
		framing.NewRequestResponseFrame(1, []byte("fake-data"), []byte("fake-metadata"), 0),
	}

	for _, it := range frames {
		n, err := common.NewUint24(it.Len())
		assert.NoError(t, err, "convert to uint24 failed")
		_, err = n.WriteTo(b)
		assert.NoError(t, err, "write length failed")
		_, err = it.WriteTo(b)
		assert.NoError(t, err, "write frame failed")
	}

	var results []core.BufferedFrame

	decoder := transport.NewLengthBasedFrameDecoder(b)
	for {
		next, err := decoder.Read()
		if err == io.EOF {
			break
		}
		assert.NoError(t, err, "decode next frame failed")
		frame, err := framing.FromBytes(next)
		assert.NoError(t, err, "read next frame failed")
		results = append(results, frame)
	}

	for i := 0; i < len(results); i++ {
		assert.Equal(t, frames[i].Header(), results[i].Header())
	}
}
