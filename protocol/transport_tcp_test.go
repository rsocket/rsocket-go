package protocol

import "testing"

func TestName(t *testing.T) {
	serv := NewTcpServerTransport(":8000")
	serv.Listen()
}
