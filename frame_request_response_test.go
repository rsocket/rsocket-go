package rsocket

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestXFrameRequestResponse_Data(t *testing.T) {
	metadata := []byte("hello")
	data := []byte("world")
	req := createRequestResponse(1234, data, metadata)
	assert.Equal(t, metadata, req.Metadata(), "metadata failed")
	assert.Equal(t, data, req.Data(), "data failed")
}
