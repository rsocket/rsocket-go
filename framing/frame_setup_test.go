package framing

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/rsocket/rsocket-go/common"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDecodeFrameSetup(t *testing.T) {
	metadata := []byte("hello")
	data := []byte("world")
	mimeMetadata, mimeData := []byte("text/plain"), []byte("text/plain")
	setup := NewFrameSetup(common.DefaultVersion, 30*time.Second, 90*time.Second, nil, mimeMetadata, mimeData, metadata, data)

	bf := &bytes.Buffer{}
	_, _ = setup.WriteTo(bf)
	fmt.Println(hex.EncodeToString(bf.Bytes()))

	assert.Equal(t, "1.0", setup.Version().String())
}
