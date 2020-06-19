package framing

import (
	"bytes"
	"log"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestDecodeFrameSetup(t *testing.T) {
	metadata := []byte("hello")
	data := []byte("world")
	mimeMetadata, mimeData := []byte("text/plain"), []byte("application/json")
	token := []byte(common.RandAlphanumeric(16))
	setup := NewSetupFrame(
		common.DefaultVersion,
		30*time.Second,
		90*time.Second,
		token,
		mimeMetadata,
		mimeData,
		data,
		metadata,
		false,
	)
	log.Println(setup)
	assert.Equal(t, "1.0", setup.Version().String())
	assert.True(t, bytes.Equal(token, setup.Token()), "bad token")
	assert.Equal(t, string(mimeMetadata), setup.MetadataMimeType(), "bad mime metadata")
	assert.Equal(t, string(mimeData), setup.DataMimeType(), "bad mime data")
	m, _ := setup.Metadata()
	assert.Equal(t, metadata, m, "bad metadata")
	assert.Equal(t, data, setup.Data(), "bad data")
}
