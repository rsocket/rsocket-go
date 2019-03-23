# rsocket-go
![logo](./logo.jpg)

[![Slack](https://img.shields.io/badge/slack-rsocket--go-blue.svg)](https://rsocket.slack.com/messages/C9VGZ5MV3)
[![GoDoc](https://godoc.org/github.com/rsocket/rsocket-go?status.svg)](https://godoc.org/github.com/rsocket/rsocket-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/rsocket/rsocket-go)](https://goreportcard.com/report/github.com/rsocket/rsocket-go)
[![License](https://img.shields.io/github/license/rsocket/rsocket-go.svg)](https://github.com/rsocket/rsocket-go/blob/master/LICENSE)
[![GitHub Release](https://img.shields.io/github/release/rsocket/rsocket-go.svg)](https://github.com/rsocket/rsocket-go/releases)

rsocket-go is an implementation of the [RSocket](http://rsocket.io/) protocol in Go. It is still under development, APIs are unstable and maybe change at any time until release of v1.0.0. **Please do not use it in a production environment**.

## Features
 - Design For Golang
 - Thin [reactive-streams](http://www.reactive-streams.org/) implementation.

## Getting started

> Start an echo server
```go
package main

import (
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

func main() {
	// Create and serve
	err := rsocket.Receive().
		Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.RSocket) rsocket.RSocket {
			// bind responder
			return rsocket.NewAbstractSocket(rsocket.RequestResponse(func(msg payload.Payload) rx.Mono {
				return rx.JustMono(msg)
			}))
		}).
		Transport("127.0.0.1:7878").
		Serve()
	panic(err)
}

```

> Connect to echo server

```go
package main

import (
	"context"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"log"
)

func main() {
	// Connect to server
	client, err := rsocket.Connect().
		SetupPayload(payload.NewString("Hello", "World")).
		Transport("127.0.0.1:7878").
		Start()
	if err != nil {
		panic(err)
	}
	defer client.Close()
	// Send request
	client.RequestResponse(payload.NewString("你好", "世界")).
		DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			log.Println("receive response:", elem)
		}).
		Subscribe(context.Background())

}

```

> NOTICE: more server examples are [Here](./example/echo/echo.go)

## Advanced

### Reactor API

`Mono` and `Flux` are two parts of Reactor API.

#### Mono

`Mono` completes successfully by emitting an element, or with an error.
Here is a tiny example:

```go
package main

import (
	"context"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

func main() {
	// Create a Mono which produce a simple payload.
	mono := rx.NewMono(func(sink rx.MonoProducer) {
		sink.Success(payload.NewString("foo", "bar"))
	})

	done := make(chan struct{})

	mono.
		DoFinally(func(ctx context.Context, st rx.SignalType) {
			close(done)
		}).
		DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			// Handle and consume payload.
			// Do something here...
		}).
		SubscribeOn(rx.ElasticScheduler()).
		Subscribe(context.Background())

	<-done
}

```

### Flux

`Flux` emits 0 to N elements, and then completes (successfully or with an error).
Here is tiny example:

``` go
package main

import (
	"context"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"time"
)

func main() {
	// Create a Flux and produce 10 elements.
	flux := rx.NewFlux(func(ctx context.Context, producer rx.Producer) {
		for i := 0; i < 10; i++ {
			producer.Next(payload.NewString("hello", time.Now().String()))
		}
		producer.Complete()
	})
	flux.
		DoOnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			// Handle and consume elements
			// Do something here...
		}).
		Subscribe(context.Background())
}

```

#### RequestN

`Flux` support **backpressure**.

You can call func `Request` in `Subscription` or use `LimitRate` before subscribe.

```go
flux.Subscribe(
    context.Background(),
    rx.OnSubscribe(func(ctx context.Context, s rx.Subscription) {
        // Init Request 1 element.
        s.Request(1)
    }),
    rx.OnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
        // Consume element, do something...

        // Request for next one manually.
        s.Request(1)
    }),
)
```

### TODO

#### Transport
 - [x] TCP
 - [ ] Websocket
 - [ ] HTTP/2

#### Duplex Socket
 - [x] MetadataPush
 - [x] RequestFNF
 - [x] RequestResponse
 - [x] RequestStream
 - [x] RequestChannel (Improve is needed.)

##### Others
 - [ ] Tuning
 - [x] Keepalive
 - [ ] Fragmentation
 - [x] Thin Reactor
 - [x] Cancel
 - [x] Error
 - [x] Flow Control: RequestN
 - [ ] Flow Control: Lease
 - [ ] Load Balance
 - [ ] Reconnect
