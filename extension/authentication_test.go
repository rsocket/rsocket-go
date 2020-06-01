package extension_test

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/extension"
	"github.com/stretchr/testify/assert"
)

func TestNewAuthentication(t *testing.T) {
	payload := []byte("foobar")

	for _, authType := range []string{"bearer", "simple"} {
		au, err := extension.NewAuthentication(authType, payload)
		assert.NoError(t, err, "create Authentication failed!")
		assert.Equal(t, authType, au.Type(), "wrong type")
		assert.Equal(t, payload, au.Payload(), "wrong payload")
		assert.Equal(t, true, au.IsWellKnown(), "well-known should be true")
		b := au.Bytes()
		au2, err := extension.ParseAuthentication(b)
		assert.NoError(t, err, "parse Authentication failed!")
		assert.Equal(t, au.Type(), au2.Type(), "authType doesn't match")
		assert.Equal(t, au.Payload(), au2.Payload(), "payload doesn't match")
	}

	_, err := extension.NewAuthentication(strings.Repeat("0", 128), payload)
	assert.True(t, extension.IsAuthTypeLengthExceed(err), "should error")
}

func TestParseAuthentication(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	input := make([]byte, 2)
	rand.Read(input)
	_, err := extension.ParseAuthentication(input)
	assert.True(t, extension.IsInvalidAuthenticationBytes(err), "should error")

}

func TestAuthentication(t *testing.T) {
	au := extension.MustNewAuthentication("simple", []byte("foobar"))
	raw := au.Bytes()
	au2, err := extension.ParseAuthentication(raw)
	assert.NoError(t, err, "bad authentication bytes")
	assert.Equal(t, au, au2, "not match")
}

func BenchmarkAuthentication_Bytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		au := extension.MustNewAuthentication("simple", []byte("foobar"))
		_ = au.Bytes()
	}
}

func BenchmarkParseAuthentication(b *testing.B) {
	au := extension.MustNewAuthentication("simple", []byte("foobar"))
	raw := au.Bytes()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extension.ParseAuthentication(raw)
	}
}
