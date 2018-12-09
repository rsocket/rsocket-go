# go-rsocket (WARNING: STILL IN DEVELOP!!!)
Unofficial [RSocket](http://rsocket.io/) in golang.

[![Build Status](https://travis-ci.org/jjeffcaii/go-rsocket.svg?branch=master)](https://travis-ci.org/jjeffcaii/go-rsocket)

## Example
```go
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
			log.Printf("SETUP: version=%s, data=%s, metadata=%s\n", setup.Version(), string(setup.Data()), string(setup.Metadata()))
			return nil
		}),
		WithFireAndForget(func(req Payload) error {
			log.Println("GOT FNF:", req)
			return nil
		}),
		WithRequestResponseHandler(func(req Payload) (res Payload, err error) {
			// just echo
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

```