package extension

import (
	"bytes"
	"testing"

	"github.com/rsocket/rsocket-go/internal/common"
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

func TestEncodeRouting_TooLarge(t *testing.T) {
	_, err := EncodeRouting(common.RandAlphanumeric(256))
	assert.Error(t, err, "should return error")
	_, err = EncodeRouting("foobar", common.RandAlphanumeric(256))
	assert.Error(t, err, "should return error")
}

func TestParseRoutingTags_Broken(t *testing.T) {
	bf := &bytes.Buffer{}
	bf.WriteByte(0xFF)
	bf.WriteString("brokenTag")

	b := bf.Bytes()
	_, err := ParseRoutingTags(b)
	assert.Error(t, err, "should return error")
}
