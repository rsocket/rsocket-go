package extension_test

import (
	"testing"

	"github.com/rsocket/rsocket-go/extension"
	"github.com/stretchr/testify/assert"
)

func TestAuthentication(t *testing.T) {
	au := extension.NewAuthentication("simple", []byte("foobar"))
	raw := au.Bytes()
	au2, err := extension.ParseAuthentication(raw)
	assert.NoError(t, err, "bad authentication bytes")
	assert.Equal(t, au, au2, "not match")
}

func BenchmarkAuthentication_Bytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		au := extension.NewAuthentication("simple", []byte("foobar"))
		_ = au.Bytes()
	}
}

func BenchmarkParseAuthentication(b *testing.B) {
	au := extension.NewAuthentication("simple", []byte("foobar"))
	raw := au.Bytes()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extension.ParseAuthentication(raw)
	}
}
