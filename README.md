# go-rsocket (WARNING: STILL IN DEVELOP!!!)
Unofficial [RSocket](http://rsocket.io/) in golang.

[![Build Status](https://travis-ci.org/jjeffcaii/go-rsocket.svg?branch=master)](https://travis-ci.org/jjeffcaii/go-rsocket)

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
    			return rsocket.NewFlux(func(ctx context.Context, emitter rsocket.Emitter) {
    				for i := 0; i < 100; i++ {
    					payload := rsocket.NewPayloadString(fmt.Sprintf("%s_%d", s, i), "")
    					emitter.Next(payload)
    				}
    				emitter.Complete()
    			})
    		}),
    		rsocket.RequestChannel(func(payloads rsocket.Publisher) rsocket.Flux {
    			return payloads.(rsocket.Flux)
    			//// echo all incoming payloads
    			//f := rsocket.NewFlux(func(emitter rsocket.Emitter) {
    			//	req.
    			//		DoFinally(func() {
    			//			emitter.Complete()
    			//		}).
    			//		SubscribeOn(rsocket.ElasticScheduler()).
    			//		Subscribe(func(item rsocket.Payload) {
    			//			emitter.Next(rsocket.NewPayload(item.Data(), item.Metadata()))
    			//		})
    			//})
    			//return f
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

    			sendingSocket.
    				RequestResponse(rsocket.NewPayloadString("ping", "From server")).
    				SubscribeOn(rsocket.ElasticScheduler()).
    				Subscribe(context.Background(), func(ctx context.Context, item rsocket.Payload) {
    					log.Println("rcv response from client:", item)
    				})

    			return responder
    		}).
    		Transport("127.0.0.1:8001").
    		Serve()
    	panic(err)
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
 - [ ] Flow Control
