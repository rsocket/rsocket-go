package payload_test

import (
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/rsocket/rsocket-go/payload"
	"github.com/stretchr/testify/assert"
)

type customPayload [2][]byte

func (c customPayload) Metadata() (metadata []byte, ok bool) {
	return c[1], len(c[1]) > 0
}

func (c customPayload) MetadataUTF8() (metadata string, ok bool) {
	return string(c[1]), len(c[1]) > 0
}

func (c customPayload) Data() []byte {
	return c[0]
}

func (c customPayload) DataUTF8() string {
	return string(c[0])
}

func TestRawPayload(t *testing.T) {
	data, metadata := []byte("hello"), []byte("world")
	p := payload.New(data, metadata)
	fmt.Println("new binary payload:", p)
	assert.Equal(t, data, p.Data(), "wrong data")
	m, ok := p.Metadata()
	assert.True(t, ok, "ok should be true")
	assert.Equal(t, metadata, m, "wrong metadata")
	assert.Equal(t, "hello", p.DataUTF8(), "wrong data string")
	mu, _ := p.MetadataUTF8()
	assert.Equal(t, "world", mu, "wrong metadata string")

	invalid := []byte{0xff, 0xfe, 0xfd}
	badPayload := payload.New(invalid, invalid)
	s := badPayload.DataUTF8()
	assert.False(t, utf8.Valid([]byte(s)))
}

func TestStrPayload(t *testing.T) {
	data, metadata := "hello", "world"
	p := payload.NewString(data, metadata)
	fmt.Println("new string payload:", p)
	assert.Equal(t, []byte(data), p.Data(), "wrong data")
	m, ok := p.Metadata()
	assert.True(t, ok, "ok should be true")
	assert.Equal(t, []byte(metadata), m, "wrong metadata")
	assert.Equal(t, data, p.DataUTF8(), "wrong data string")
	mu, _ := p.MetadataUTF8()
	assert.Equal(t, metadata, mu, "wrong metadata string")
}

func TestClone(t *testing.T) {
	check := func(p1 payload.Payload) {
		p2 := payload.Clone(p1)
		assert.Equal(t, p1.Data(), p2.Data(), "bad data")
		m1, _ := p1.Metadata()
		m2, _ := p2.Metadata()
		assert.Equal(t, m1, m2, "bad metadata")
	}

	check(payload.NewString("hello", "world"))
	check(payload.New([]byte("hello"), []byte("world")))
	// Check custom payload
	custom := customPayload([2][]byte{[]byte("hello"), []byte("world")})
	check(custom)

	// Check clone nil
	nilCloned := payload.Clone(nil)
	assert.Nil(t, nilCloned, "should return nil")
}

func TestNewFile(t *testing.T) {
	p := payload.MustNewFile("/etc/hosts", nil)
	assert.NotEmpty(t, p.Data(), "empty data")
	_, err := payload.NewFile("/not/existing", nil)
	assert.Error(t, err, "should return error")

	func() {
		defer func() {
			e := recover()
			assert.Error(t, e.(error), "should panic error")
		}()
		payload.MustNewFile("/not/existing", nil)
	}()
}

func TestEmpty(t *testing.T) {
	p := payload.Empty()
	assert.Nil(t, p.Data())
}
