package framing

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/stretchr/testify/assert"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestPrintFrame(t *testing.T) {
	mime := []byte("fake mime")
	data := make([]byte, 100)
	metadata := make([]byte, 50)
	rand.Read(data)
	rand.Read(metadata)
	data = append([]byte("fake data"), data...)
	metadata = append([]byte("fake metadata"), metadata...)
	for _, f := range []core.Frame{
		NewCancelFrame(_sid),
		NewPayloadFrame(_sid, data, metadata, core.FlagComplete|core.FlagNext|core.FlagFollow),
		NewRequestResponseFrame(_sid, data, metadata, core.FlagComplete|core.FlagNext|core.FlagFollow),
		NewMetadataPushFrame(metadata),
		NewFireAndForgetFrame(_sid, data, metadata, core.FlagComplete|core.FlagNext|core.FlagFollow),
		NewRequestStreamFrame(_sid, 1, data, metadata, core.FlagComplete|core.FlagNext|core.FlagFollow),
		NewRequestChannelFrame(_sid, 1, data, metadata, core.FlagComplete|core.FlagNext|core.FlagFollow),
		NewSetupFrame(core.DefaultVersion, 30*time.Second, 90*time.Second, nil, mime, mime, data, metadata, false),
	} {
		tryPrintFrame(t, f)
	}
}

func tryPrintFrame(t *testing.T, f core.Frame) {
	s := PrintFrame(f)
	assert.True(t, len(s) > 0)
	fmt.Println(s)
}
