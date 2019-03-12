[![Slack](https://img.shields.io/badge/slack-rsocket--go-blue.svg)](https://rsocket.slack.com/messages/C9VGZ5MV3)
[![GoDoc](https://godoc.org/github.com/rsocket/rsocket-go?status.svg)](https://godoc.org/github.com/rsocket/rsocket-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/rsocket/rsocket-go)](https://goreportcard.com/report/github.com/rsocket/rsocket-go)
[![License](https://img.shields.io/github/license/rsocket/rsocket-go.svg)](https://github.com/rsocket/rsocket-go/blob/master/LICENSE)
[![GitHub Release](https://img.shields.io/github/release/rsocket/rsocket-go.svg)](https://github.com/rsocket/rsocket-go/releases)

# go-rsocket (WARNING: STILL IN DEVELOP!!! APIs will change at any time until release of v1.0.0)
[RSocket](http://rsocket.io/) in golang.

## Features
 - Design For Golang
 - Thin [reactive-streams](http://www.reactive-streams.org/) implementation.

## Reactor Start Up

> Here's basic Reactor Flux example:

``` go
	f := NewFlux(func(ctx context.Context, producer Producer) {
		for i := 0; i < 3; i++ {
			producer.Next(NewPayloadString(fmt.Sprintf("foo%d", i), fmt.Sprintf("bar%d", i)))
		}
		producer.Complete()
	})
	f.
		DoFinally(func(ctx context.Context, st SignalType) {
			log.Println("doFinally:", st)
		}).
		DoOnNext(func(ctx context.Context, s Subscription, payload Payload) {
			log.Println("doNext:", payload)
		}).
		DoAfterNext(func(ctx context.Context, payload Payload) {
			payload.Release()
		}).
		Subscribe(context.Background())

    // Or you can use varargs with wrapper of `OnSubscribe`,`OnNext`,`OnComplete` and `OnError`
    f.Subscribe(context.Background(), OnNext(func(ctx context.Context, s Subscription, payload Payload) {
		// Do something

        // Cancel it
        // s.Cancel()

        // Also you can use Request for flow control
        // s.Request(1)
	}))

```

### Example

#### Receive

```go
package main

import (
	"fmt"
	"log"
	"context"

	"github.com/rsocket/rsocket-go"
)

func main() {
	responder := rsocket.NewAbstractSocket(
    		rsocket.MetadataPush(func(payload rsocket.Payload) {
    			log.Println("GOT METADATA_PUSH:", payload)
    		}),
    		rsocket.FireAndForget(func(payload rsocket.Payload) {
    			log.Println("GOT FNF:", payload)
    		}),
    		rsocket.RequestResponse(func(payload rsocket.Payload) rsocket.Mono {
    			// just echo
    			return rsocket.JustMono(payload)
    		}),
    		rsocket.RequestStream(func(payload rsocket.Payload) rsocket.Flux {
    			s := string(payload.Data())
    			return rsocket.NewFlux(func(ctx context.Context, emitter rsocket.Producer) {
    				for i := 0; i < 100; i++ {
    					payload := rsocket.NewPayloadString(fmt.Sprintf("%s_%d", s, i), "")
    					emitter.Next(payload)
    				}
    				emitter.Complete()
    			})
    		}),
    		rsocket.RequestChannel(func(payloads rsocket.Publisher) rsocket.Flux {
    			// eg: copy all incoming payloads
    			return rsocket.NewFlux(func(ctx context.Context, emitter rsocket.Producer) {
    				payloads.Subscribe(ctx, rsocket.OnNext(func(ctx context.Context, s rsocket.Subscription, payload rsocket.Payload) {
    					emitter.Next(rsocket.NewPayload(payload.Data(), payload.Metadata()))    					
    				}), rsocket.OnComplete(func(ctx context.Context) {
    				    emitter.Complete()
    				}))
    			})
    			// Or just Echo
    			// return payloads.(rsocket.Flux)
    		}),
    	)
    	err := rsocket.Receive().
    		Acceptor(func(setup rsocket.SetupPayload, sendingSocket rsocket.RSocket) rsocket.RSocket {
    			log.Println("SETUP BEGIN:----------------")
    			log.Println("maxLifeTime:", setup.MaxLifetime())
    			log.Println("keepaliveInterval:", setup.TimeBetweenKeepalive())
    			log.Println("dataMimeType:", setup.DataMimeType())
    			log.Println("metadataMimeType:", setup.MetadataMimeType())
    			log.Println("data:", string(setup.Data()))
    			log.Println("metadata:", string(setup.Metadata()))
    			log.Println("SETUP END:----------------")

    			sendingSocket.RequestResponse(rsocket.NewPayloadString("ping", "From server")).
    				DoOnSuccess(func(ctx context.Context, s rsocket.Subscription, payload rsocket.Payload) {
    					log.Println("rcv response from client:", payload)
    				}).
    				SubscribeOn(rsocket.ElasticScheduler()).
    				Subscribe(context.Background())

    			return responder
    		}).
    		Transport("127.0.0.1:8001").
    		Serve()
    	if err != nil {
    		panic(err)	
    	}
}

```

#### Connect

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rsocket/rsocket-go"
)

type RSocket = rsocket.RSocket
type Payload = rsocket.Payload
type Mono = rsocket.Mono
type Flux = rsocket.Flux
type Emitter = rsocket.Emitter

func main() {

	socket, err := rsocket.Connect().
		SetupPayload(rsocket.NewPayloadString("hello", "world")).
		MetadataMimeType("application/json").
		DataMimeType("application/json").
		KeepAlive(3*time.Second, 2*time.Second, 3).
		Acceptor(func(socket RSocket) RSocket {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(payload Payload) Mono {
					return rsocket.JustMono(rsocket.NewPayloadString("foo", "bar"))
				}),
			)
		}).
		Transport("127.0.0.1:8001").
		Start()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = socket.Close()
	}()

	socket.RequestResponse(rsocket.NewPayloadString("see", "you")).
		Subscribe(context.Background(), func(ctx context.Context, item Payload) {
			log.Println("WAHAHA:", item)
		})

	socket.RequestStream(rsocket.NewPayloadString("aaa", "bbb")).
		DoFinally(func(ctx context.Context) {
			log.Println("finish")
		}).
		Subscribe(context.Background(), func(ctx context.Context, item Payload) {
			log.Println("stream:", item)
		})

	socket.
		RequestChannel(rsocket.NewFlux(func(ctx context.Context, emitter Emitter) {
			for i := 0; i < 5; i++ {
				emitter.Next(rsocket.NewPayloadString(fmt.Sprintf("hello_%d", i), "from golang"))
			}
			emitter.Complete()
		})).
		Subscribe(context.Background(), func(ctx context.Context, item Payload) {
			log.Println("channel:", item)
		})
}

```

### TODO

#### Transport
 - [x] TCP
 - [ ] Websocket

#### Duplex Socket (Include basic features. Improvements are needed.)
 - [x] MetadataPush
 - [x] RequestFNF
 - [x] RequestResponse
 - [x] RequestStream
 - [x] RequestChannel

##### Others
 - [ ] Tuning
 - [x] Keepalive
 - [ ] Fragmentation
 - [ ] Full Reactor Support
 - [x] Cancel
 - [x] Error
 - [x] Flow Control
