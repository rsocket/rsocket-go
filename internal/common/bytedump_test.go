package common

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestAppendPrettyHexDump(t *testing.T) {
	b := make([]byte, 100)
	rand.Read(b)
	s := PrettyHexDump(b)
	assert.NotEmpty(t, s, "should not return empty string")
}

func TestAppendPrettyHexDump_Empty(t *testing.T) {
	s := PrettyHexDump(nil)
	assert.Empty(t, s, "should return empty string")
	s = PrettyHexDump([]byte{})
	assert.Empty(t, s, "should return empty string")
}

func TestAppendPrettyHexDump_Big(t *testing.T) {
	// 16MB
	b := make([]byte, 16*1024*1024)
	rand.Read(b)
	sb := &strings.Builder{}
	AppendPrettyHexDump(sb, b)
	assert.NotEmpty(t, sb.String(), "should not return empty string")
}
