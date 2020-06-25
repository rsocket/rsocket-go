package core_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/rsocket/rsocket-go/core"
	"github.com/stretchr/testify/assert"
)

func BenchmarkVersion_String(b *testing.B) {
	v := core.NewVersion(2, 3)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.String()
	}
}

func TestVersion(t *testing.T) {
	var (
		major uint16 = 2
		minor uint16 = 1
	)
	v := core.NewVersion(major, minor)
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

func TestVersion_Equals(t *testing.T) {
	v1 := core.NewVersion(1, 3)
	v2 := core.NewVersion(1, 3)
	v3 := core.NewVersion(3, 1)
	v4 := core.NewVersion(1, 2)

	assert.True(t, v1.Equals(v2))
	assert.True(t, v2.Equals(v1))
	assert.False(t, v1.Equals(v3))
	assert.False(t, v1.Equals(v4))
}

func TestVersion_GreaterThan(t *testing.T) {
	v1 := core.NewVersion(1, 3)
	v2 := core.NewVersion(1, 3)
	v3 := core.NewVersion(3, 1)
	v4 := core.NewVersion(1, 2)
	assert.False(t, v1.GreaterThan(v2))
	assert.False(t, v1.GreaterThan(v3))
	assert.True(t, v1.GreaterThan(v4))
}

func TestVersion_LessThan(t *testing.T) {
	v1 := core.NewVersion(1, 3)
	v2 := core.NewVersion(1, 3)
	v3 := core.NewVersion(3, 1)
	v4 := core.NewVersion(1, 2)
	assert.False(t, v1.LessThan(v2))
	assert.True(t, v1.LessThan(v3))
	assert.False(t, v1.LessThan(v4))
}

func checkBytes(t *testing.T, b []byte, expectMajor, expectMinor uint16) {
	assert.Equal(t, 4, len(b), "wrong version bytes")
	major := binary.BigEndian.Uint16(b[:2])
	minor := binary.BigEndian.Uint16(b[2:])
	assert.Equal(t, expectMajor, major, "wrong major version")
	assert.Equal(t, expectMinor, minor, "wrong minor version")
}
