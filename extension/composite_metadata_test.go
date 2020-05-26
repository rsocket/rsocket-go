package extension

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompositeMetadataBuilder_Build(t *testing.T) {
	cm, err := NewCompositeMetadataBuilder().
		PushString("application/custom", "not well").
		PushString("text/plain", "text").
		PushWellKnownString(ApplicationJSON, `{"hello":"world"}`).
		Build()
	assert.NoError(t, err, "build composite metadata failed")
	scanner := cm.Scanner()
	for scanner.Scan() {
		mimeType, metadata, err := scanner.Metadata()
		assert.NoError(t, err, "scan metadata failed")
		fmt.Println("mimeType:", mimeType, "metadata:", string(metadata))
	}
}
