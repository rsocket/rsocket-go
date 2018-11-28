package example

import (
	"context"
	"github.com/jjeffcaii/go-rsocket"
	"github.com/jjeffcaii/go-rsocket/protocol"
	"testing"
)

func TestTcpServerTransport(t *testing.T) {
	trans := protocol.NewTcpServerTransport(":8000")
	server, err := rsocket.Builder().
		Transport(trans).
		Acceptor(func(setup *rsocket.SetupPayload, sendingSocket *rsocket.RSocket) (err error) {
			return nil
		}).
		Build()
	if err != nil {
		t.Error(err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Error(err)
	}
}
