# go-rsocket (WARNING: STILL IN DEVELOP!!!)
Unofficial [RSocket](http://rsocket.io/) in golang.

[![Build Status](https://travis-ci.org/jjeffcaii/go-rsocket.svg?branch=master)](https://travis-ci.org/jjeffcaii/go-rsocket)

### Example

#### Server-Side

```go
package main

import (
	"fmt"
	"log"
	
	"github.com/jjeffcaii/go-rsocket"
)

func main() {
    server, err := rsocket.NewServer(
		rsocket.WithTCPServerTransport("127.0.0.1:8000"),
		rsocket.WithAcceptor(func(setup rsocket.SetupPayload, rs *rsocket.RSocket) (err error) {
			
			log.Println("maxLifeTime:", setup.MaxLifetime())
			log.Println("keepaliveInterval:", setup.TimeBetweenKeepalive())
			log.Println("dataMimeType:", setup.DataMimeType())
			log.Println("metadataMimeType:", setup.MetadataMimeType())
			log.Println("data:", string(setup.Data()))
			log.Println("metadata:", string(setup.Metadata()))

			rs.HandleMetadataPush(func(metadata []byte) {
				log.Println("GOT METADATA_PUSH:", string(metadata))
			})

			rs.HandleFireAndForget(func(req rsocket.Payload) {
				log.Println("GOT FNF:", req)
			})

			rs.HandleRequestResponse(func(req rsocket.Payload) (res rsocket.Payload, err error) {
				// just echo
				return req, nil
			})
			
			rs.HandleRequestStream(func(req rsocket.Payload) rsocket.Flux {
				d := string(req.Data())
				return rsocket.NewFlux(func(emitter rsocket.Emitter) {
					for i := 0; i < 100; i++ {
						payload := rsocket.NewPayload([]byte(fmt.Sprintf("%s_%d", d, i)), nil)
						emitter.Next(payload)
					}
					emitter.Complete()
				})
			})

			rs.HandleRequestChannel(func(req rsocket.Flux) (res rsocket.Flux) {
				// echo all incoming payloads
				
				// Just echo
				return req
				
				// Or do something as you wish
				/*
				return rsocket.NewFlux(func(emitter rsocket.Emitter) {
					req.
						DoFinally(func() {
							emitter.Complete()
						}).
						SubscribeOn(rsocket.ElasticScheduler()).
						Subscribe(func(item rsocket.Payload) {
							emitter.Next(rsocket.NewPayload(item.Data(), item.Metadata()))
						})
				})
				*/
			})
			return
		}),
	)
    if err != nil {
    	panic(err)
    }
    defer func() {
        _ = server.Close()
    }()
    server.Serve()
}

```

#### Client-Side

> WARNNING: Only RequestResponse is supported now. Others are in process of development.

```go
package main

import(
	"context"
	"fmt"
	"log"
	"time"
	
	"github.com/jjeffcaii/go-rsocket"
)


func main() {
    ctx, cancel := context.WithCancel(context.Background())

    defer func() {
        cancel()
    }()

    cli, err := rsocket.NewClient(
        rsocket.WithTCPTransport("127.0.0.1", 8000),
        rsocket.WithSetupPayload([]byte("你好"), []byte("世界")),
        rsocket.WithKeepalive(2*time.Second, 3*time.Second, 3),
        rsocket.WithMetadataMimeType("application/binary"),
        rsocket.WithDataMimeType("application/binary"),
    )
    if err != nil {
        panic(err)
    }
    defer func() {
    	_ = cli.Close()
    }()
    if err := cli.Start(ctx); err != nil {
        panic(err)
    }
    done := make(chan struct{})
    err = cli.RequestResponse([]byte("hello"), []byte("world"), func(res rsocket.Payload, err error) {
        defer close(done)
        if err != nil {
            log.Println("Oops....", err)
        } else {
            log.Println("RESPONSE:", res)
        }
    })
    if err != nil {
        panic(err)
    }
    <-done
 }
```

### Todo

#### Transport
 - [x] TCP
 - [ ] Websocket

#### Responder

##### Server-Side
 - [x] MetadataPush
 - [x] RequestFNF
 - [x] RequestResponse
 - [x] RequestStream
 - [x] RequestChannel
 
##### Client-Side
 - [ ] MetadataPush
 - [ ] RequestFNF
 - [ ] RequestResponse
 - [ ] RequestStream
 - [ ] RequestChannel

#### Requester

##### Server-Side
 - [ ] MetadataPush
 - [ ] RequestFNF
 - [ ] RequestResponse
 - [ ] RequestStream
 - [ ] RequestChannel
 
##### Client-Side
 - [ ] MetadataPush
 - [ ] RequestFNF
 - [x] RequestResponse
 - [ ] RequestStream
 - [ ] RequestChannel

##### Others
 - [ ] Keepalive
 - [ ] Fragmentation
 - [ ] Full Reactor Support
 - [ ] Cancel
 - [ ] Error
 - [ ] Flow Control
