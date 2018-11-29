package rsocket

import (
	"context"
	"fmt"
	"log"
	"testing"
)

func TestRSocketServer_Start(t *testing.T) {
	server, err := NewServer(
		WithTransportTCP("127.0.0.1:8000"),
		WithAcceptor(func(setup *SetupPayload, rs *RSocket) (err error) {
			log.Println("SETUP:", setup)
			return nil
		}),
		WithRequestResponseHandler(func(request *Payload) (reponse *Payload, err error) {
			log.Println("RQ:", request)
			foo := string(request.Data)
			if foo == "ping" {
				return &Payload{
					Data:     []byte("pong"),
					Metadata: request.Metadata,
				}, nil
			}
			return nil, fmt.Errorf("invalid data: %s", foo)
		}),
	)
	if err != nil {
		t.Error(err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Error(err)
	}
}
