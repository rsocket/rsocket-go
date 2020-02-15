package extension

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRoutingTags(t *testing.T) {
	raw, err := EncodeRouting("/foo", "/bar", "/foo/bar")
	assert.NoError(t, err, "encode routing failed")
	tags, err := ParseRoutingTags(raw)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 3, len(tags))
	assert.Equal(t, "/foo", tags[0])
	assert.Equal(t, "/bar", tags[1])
	assert.Equal(t, "/foo/bar", tags[2])
}
