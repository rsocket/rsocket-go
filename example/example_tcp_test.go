package example

import (
	"github.com/jjeffcaii/go-rsocket/protocol"
	"testing"
)

func TestTcpServerTransport(t *testing.T) {
	serv := protocol.NewTcpServerTransport(":8000")
	serv.Listen()
}
