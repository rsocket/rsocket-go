package rs

import (
	"bufio"
	"github.com/jjeffcaii/go-rsocket/protocol"
	"net"
)

type RSocketServer struct {
	transport protocol.Transport
}

func (p *RSocketServer) Start() error {
	listener, err := net.Listen("tcp", ":1943")
	if err != nil {
		panic(err)
	}

	for {
		select {
		default:
			c, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			go dispatch(c)
		}
	}
}

func (p *RSocketServer) dispatch() {
	reader := bufio.NewReader(c)
	writer := bufio.NewWriter(c)
}

func NewRsocketServer() (*RSocketServer, error) {
	return &RSocketServer{}, nil
}
