package framing_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
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
	for _, f := range []core.WriteableFrame{
		framing.NewCancelFrame(_sid),
		framing.NewPayloadFrame(_sid, data, metadata, core.FlagComplete|core.FlagNext|core.FlagFollow),
		framing.NewRequestResponseFrame(_sid, data, metadata, core.FlagComplete|core.FlagNext|core.FlagFollow),
		framing.NewMetadataPushFrame(metadata),
		framing.NewFireAndForgetFrame(_sid, data, metadata, core.FlagComplete|core.FlagNext|core.FlagFollow),
		framing.NewRequestStreamFrame(_sid, 1, data, metadata, core.FlagComplete|core.FlagNext|core.FlagFollow),
		framing.NewRequestChannelFrame(_sid, 1, data, metadata, core.FlagComplete|core.FlagNext|core.FlagFollow),
		framing.NewSetupFrame(core.DefaultVersion, 30*time.Second, 90*time.Second, nil, mime, mime, data, metadata, false),
	} {
		tryPrintFrame(t, f)
	}
}

func tryPrintFrame(t *testing.T, f core.WriteableFrame) {
	s := framing.PrintFrame(f)
	assert.True(t, len(s) > 0)
	fmt.Println(s)
}
