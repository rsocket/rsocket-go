package rsocket_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rsocket/rsocket-go"
	"github.com/stretchr/testify/assert"
)

func TestUnix(t *testing.T) {
	sockFile := fmt.Sprintf("%s/test-rsocket-%s.sock", strings.TrimRight(os.TempDir(), "/"), uuid.New().String())
	defer os.Remove(sockFile)
	u := rsocket.Unix().Path(sockFile).Build()
	assert.NotNil(t, u)
	_, err := u.Server()(context.Background())
	assert.NoError(t, err)
}

func TestTcp(t *testing.T) {
	rsocket.Tcp().HostAndPort("127.0.0.1", 7878).Build()
}

func TestWebsocket(t *testing.T) {
	rsocket.Websocket()
}
