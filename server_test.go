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
		WithAcceptor(func(setup SetupPayload, rs *RSocket) (err error) {
			log.Println("SETUP:", setup)
			return nil
		}),
		WithRequestResponseHandler(func(req Payload) (res Payload, err error) {
			return req, nil
		}),
		WithRequestStreamHandler(func(req Payload, emitter Emitter) {
			totals := 1000
			for i := 0; i < totals; i++ {
				payload := CreatePayloadString(fmt.Sprintf("%d", i), "")
				if err := emitter.Next(payload); err != nil {
					log.Println("process stream failed:", err)
				}
			}
			payload := CreatePayloadString(fmt.Sprintf("%d", totals), "")
			if err := emitter.Complete(payload); err != nil {
				log.Println("process stream failed:", err)
			}
		}),
	)
	if err != nil {
		t.Error(err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Error(err)
	}
}

type CCC []byte

func (p CCC) String() string {
	return string(p)
}

func TestFoobar(t *testing.T) {
	bs := []byte("abcdefg")
	cc := CCC(bs)
	println("cc:", cc.String())
	s := string(bs)
	println(s)
	sp := bs[:4]
	bs[1] = '?'
	println(s)
	println(string(bs))
	println(string(sp))
	println("cc:", cc.String())
}
