package extension

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseRoutingTags(t *testing.T) {
	bf := &bytes.Buffer{}
	bf.WriteByte(4)
	bf.WriteString("/foo")
	bf.WriteByte(4)
	bf.WriteString("/bar")
	bf.WriteByte(8)
	bf.WriteString("/foo/bar")

	bs := bf.Bytes()
	tags, err := ParseRoutingTags(bs)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 3, len(tags))
	assert.Equal(t, "/foo", tags[0])
	assert.Equal(t, "/bar", tags[1])
	assert.Equal(t, "/foo/bar", tags[2])
}
