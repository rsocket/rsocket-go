package rsocket_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rsocket/rsocket-go"
	"github.com/stretchr/testify/assert"
)

var fakeSockFile string
var fakeTlsConfig = &tls.Config{
	InsecureSkipVerify: true,
}

func init() {
	fakeSockFile = fmt.Sprintf("%s/test-rsocket-%s.sock", strings.TrimRight(os.TempDir(), "/"), uuid.New().String())
}

func TestUnixServer(t *testing.T) {
	defer os.Remove(fakeSockFile)
	u := rsocket.UnixServer().SetPath(fakeSockFile).Build()
	assert.NotNil(t, u)
	_, err := u(context.Background())
	assert.NoError(t, err)
}

func TestUnixClient(t *testing.T) {
	assert.NotPanics(t, func() {
		rsocket.UnixClient().SetPath(fakeSockFile).Build()
	})
}

func TestTcpClient(t *testing.T) {
	assert.NotPanics(t, func() {
		rsocket.TCPClient().
			SetAddr(":7878").
			SetHostAndPort("127.0.0.1", 7878).
			SetTLSConfig(fakeTlsConfig).
			Build()
	})
}

func TestTcpServerBuilder(t *testing.T) {
	assert.NotPanics(t, func() {
		rsocket.TCPServer().SetAddr(":7878").Build()
		rsocket.TCPServer().SetHostAndPort("127.0.0.1", 7878).SetTLSConfig(fakeTlsConfig).Build()
	})
}

func TestWebsocketClient(t *testing.T) {
	assert.NotPanics(t, func() {
		h := make(http.Header)
		h.Set("x-foo-bar", "qux")
		rsocket.WebsocketClient().
			SetURL("ws://127.0.0.1:8080/fake/path").
			SetHeader(h).
			SetTLSConfig(fakeTlsConfig).
			Build()
	})
}

func TestWebsocketServer(t *testing.T) {
	assert.NotPanics(t, func() {
		tp := rsocket.WebsocketServer().
			SetAddr(":7878").
			SetPath("/fake").
			SetTLSConfig(fakeTlsConfig).
			Build()
		assert.NotNil(t, tp)
	})
}
