package common_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/stretchr/testify/assert"
)

func BenchmarkVersion_String(b *testing.B) {
	v := common.NewVersion(2, 3)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.String()
	}
}

func TestVersion(t *testing.T) {
	var (
		major uint16 = 2
		minor uint16 = 1
	)
	v := common.NewVersion(major, minor)
	assert.Equal(t, "2.1", v.String())
	assert.Equal(t, uint16(2), v.Major(), "wrong major version")
	assert.Equal(t, uint16(1), v.Minor(), "wrong minor version")
	checkBytes(t, v.Bytes(), 2, 1)
	w := bytes.Buffer{}
	n, err := v.WriteTo(&w)
	assert.NoError(t, err, "write version failed")
	assert.Equal(t, int64(4), n, "wrong wrote bytes length")
	checkBytes(t, v.Bytes(), 2, 1)
}

func checkBytes(t *testing.T, b []byte, expectMajor, expectMinor uint16) {
	assert.Equal(t, 4, len(b), "wrong version bytes")
	major := binary.BigEndian.Uint16(b[:2])
	minor := binary.BigEndian.Uint16(b[2:])
	assert.Equal(t, expectMajor, major, "wrong major version")
	assert.Equal(t, expectMinor, minor, "wrong minor version")
}
