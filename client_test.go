package rsocket

import (
	"context"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestClient_Connect(t *testing.T) {
	cli, err := NewClient(
		WithKeepalive(2*time.Second, 3*time.Second, 3),
		WithTCPTransport("127.0.0.1", 8000),
		WithSetupPayload([]byte("hello"), []byte("world")),
	)
	assert.NoError(t, err)
	defer func() {
		_ = cli.Close()
	}()
	err = cli.Start(context.Background())
	assert.NoError(t, err, "connect rsocket failed")
	time.Sleep(1 * time.Hour)
}

func TestClient_MetadataPush(t *testing.T) {
	cli, err := NewClient(
		WithTCPTransport("127.0.0.1", 8000),
		WithSetupPayload([]byte("hello"), []byte("world")),
	)
	assert.NoError(t, err)
	defer func() {
		_ = cli.Close()
	}()
	err = cli.Start(context.Background())
	assert.NoError(t, err, "connect rsocket failed")
	err = cli.MetadataPush([]byte("hello world!"))
	assert.NoError(t, err, "send metadata_push failed")
}

func TestClient_FireAndForget(t *testing.T) {
	cli, err := NewClient(
		WithTCPTransport("127.0.0.1", 8000),
		WithSetupPayload([]byte("hello"), []byte("world")),
	)
	assert.NoError(t, err)
	defer func() {
		_ = cli.Close()
	}()
	err = cli.Start(context.Background())
	assert.NoError(t, err, "connect rsocket failed")
	err = cli.FireAndForget([]byte("hello"), []byte("world"))
	assert.NoError(t, err, "send fnf failed")
}

func TestClient_RequestResponse(t *testing.T) {
	cli, err := NewClient(
		WithTCPTransport("127.0.0.1", 8000),
		WithSetupPayload([]byte("hello"), []byte("world")),
	)
	assert.NoError(t, err)
	defer func() {
		_ = cli.Close()
	}()
	err = cli.Start(context.Background())
	assert.NoError(t, err, "connect rsocket failed")
	wg := sync.WaitGroup{}
	wg.Add(1)
	err = cli.RequestResponse([]byte("foo"), []byte("bar"), func(res Payload, err error) {
		defer wg.Done()
		assert.NoError(t, err)
		assert.Equal(t, "foo", string(res.Data()))
		assert.Equal(t, "bar", string(res.Metadata()))
	})
	assert.NoError(t, err)
	wg.Wait()
}
