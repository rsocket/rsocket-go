package protocol

import "testing"

func TestName(t *testing.T) {
	serv := NewTcpServerTransport(":6789")
	serv.Listen()
}
