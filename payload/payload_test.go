package payload

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayload_new(t *testing.T) {
	data, metadata := "hello", "world"
	p1 := New([]byte(data), []byte(metadata))

	assert.Equal(t, data, p1.DataUTF8(), "bad data")
	metadata2, ok := p1.MetadataUTF8()
	assert.True(t, ok, "bad metadata")
	assert.Equal(t, metadata, metadata2, "bad metadata")

	p1 = New([]byte(data), nil)
	metadata2, ok = p1.MetadataUTF8()
	assert.False(t, ok)
	assert.Equal(t, "", metadata2)
}

func TestNewFile(t *testing.T) {
	pl, err := NewFile("/etc/hosts", nil)
	assert.NoError(t, err, "bad file")
	fmt.Print(pl.DataUTF8())
}
