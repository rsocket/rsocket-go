package benchmark

import (
	"bytes"
	"fmt"
	"github.com/rsocket/rsocket-go"
	"log"
)

func createClient(host string, port int) rsocket.ClientSocket {
	client, err := rsocket.Connect().
		SetupPayload(rsocket.NewPayloadString("你好", "世界")).
		Acceptor(func(socket rsocket.RSocket) rsocket.RSocket {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(payload rsocket.Payload) rsocket.Mono {
					log.Println("rcv reqresp from server:", payload)
					if bytes.Equal(payload.Data(), []byte("ping")) {
						return rsocket.JustMono(rsocket.NewPayloadString("pong", "from client"))
					}
					return rsocket.JustMono(payload)
				}),
			)
		}).
		Transport(fmt.Sprintf("%s:%d", host, port)).
		Start()
	if err != nil {
		panic(err)
	}
	return client
}
