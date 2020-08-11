package rsocket_test

import (
	"context"
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

func init() {
	fmt.Println(os.TempDir())
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
		rsocket.TcpClient().
			SetAddr(":7878").
			SetHostAndPort("127.0.0.1", 7878).
			Build()
	})
}

func TestTcpServerBuilder(t *testing.T) {
	assert.NotPanics(t, func() {
		rsocket.TcpServer().SetAddr(":7878").Build()
	})
}

func TestWebsocketClient(t *testing.T) {
	assert.NotPanics(t, func() {
		h := make(http.Header)
		h.Set("x-foo-bar", "qux")
		rsocket.WebsocketClient().
			SetUrl("ws://127.0.0.1:8080/fake/path").
			SetHeader(h).
			Build()
	})
}

func TestWebsocketServer(t *testing.T) {
	assert.NotPanics(t, func() {
		tp := rsocket.WebsocketServer().
			SetAddr(":7878").
			SetPath("/fake").
			Build()
		assert.NotNil(t, tp)
	})
}