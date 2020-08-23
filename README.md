# rsocket-go
![logo](./logo.jpg)

[![Build Status](https://travis-ci.com/rsocket/rsocket-go.svg?branch=master)](https://travis-ci.com/rsocket/rsocket-go)
[![Coverage Status](https://coveralls.io/repos/github/rsocket/rsocket-go/badge.svg?branch=master)](https://coveralls.io/github/rsocket/rsocket-go?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/rsocket/rsocket-go)](https://goreportcard.com/report/github.com/rsocket/rsocket-go)
[![Slack](https://img.shields.io/badge/slack-rsocket--go-blue.svg)](https://rsocket.slack.com/messages/C9VGZ5MV3)
[![GoDoc](https://godoc.org/github.com/rsocket/rsocket-go?status.svg)](https://godoc.org/github.com/rsocket/rsocket-go)
[![License](https://img.shields.io/github/license/rsocket/rsocket-go.svg)](https://github.com/rsocket/rsocket-go/blob/master/LICENSE)
[![GitHub Release](https://img.shields.io/github/release-pre/rsocket/rsocket-go.svg)](https://github.com/rsocket/rsocket-go/releases)

> rsocket-go is an implementation of the [RSocket](http://rsocket.io/) protocol in Go.
<br>🚧🚧🚧 ***IT IS UNDER ACTIVE DEVELOPMENT, APIs are unstable and maybe change at any time until release of v1.0.0.***
<br>⚠️⚠️⚠️ ***DO NOT USE IN ANY PRODUCTION ENVIRONMENT!!!***

## Features
 - Design For Golang.
 - Thin [reactive-streams](http://www.reactive-streams.org/) implementation.
 - Simulate Java SDK API.
 - Fast CLI (Compatible with [https://github.com/rsocket/rsocket-cli](https://github.com/rsocket/rsocket-cli/)).
   - Installation: `go get github.com/rsocket/rsocket-go/cmd/rsocket-cli`
   - Example: `rsocket-cli --request -i hello_world --setup setup_me tcp://127.0.0.1:7878`

## Install

```shell
$ go get -u github.com/rsocket/rsocket-go
```

## Quick Start

> Start an echo server

```go
package main

import (
	"context"
	"log"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/mono"
)

func main() {
	err := rsocket.Receive().
		Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			// bind responder
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
					return mono.Just(msg)
				}),
			), nil
		}).
		Transport(rsocket.TcpServer().SetAddr(":7878").Build()).
		Serve(context.Background())
	log.Fatalln(err)
}

```

> Connect to echo server

```go
package main

import (
	"context"
	"log"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/payload"
)

func main() {
	// Connect to server
	cli, err := rsocket.Connect().
		SetupPayload(payload.NewString("Hello", "World")).
		Transport(rsocket.TcpClient().SetHostAndPort("127.0.0.1", 7878).Build()).
		Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer cli.Close()
	// Send request
	result, err := cli.RequestResponse(payload.NewString("你好", "世界")).Block(context.Background())
	if err != nil {
		panic(err)
	}
	log.Println("response:", result)
}
```

> NOTICE: more server examples are [Here](examples/echo/echo.go)

## Advanced

### Load Balance

Basic load balance feature, see [here](./balancer).

### Reactor API

`Mono` and `Flux` are two parts of Reactor API. They are based on my another project [reactor-go](https://github.com/jjeffcaii/reactor-go/).

#### Mono

`Mono` completes successfully by emitting an element, or with an error.
Here is a tiny example:

```go
package main

import (
	"context"
	"fmt"

	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/mono"
)

func main() {
	// Create a Mono using Just.
	m := mono.Just(payload.NewString("Hello World!", "text/plain"))

	// More create
	//m := mono.Create(func(i context.Context, sink mono.Sink) {
	//	sink.Success(payload.NewString("Hello World!", "text/plain"))
	//})

	done := make(chan struct{})

	m.
		DoFinally(func(s rx.SignalType) {
			close(done)
		}).
		DoOnSuccess(func(input payload.Payload) error {
			// Handle and consume payload.
			// Do something here...
			fmt.Println("bingo:", input)
			return nil
		}).
		SubscribeOn(scheduler.Parallel()).
		Subscribe(context.Background())

	<-done
}
```

### Flux

`Flux` emits 0 to N elements, and then completes (successfully or with an error).
Here is tiny example:

```go
package main

import (
	"context"
	"fmt"

	"github.com/rsocket/rsocket-go/extension"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
)

func main() {
	// Create a Flux and produce 10 elements.
	f := flux.Create(func(ctx context.Context, sink flux.Sink) {
		for i := 0; i < 10; i++ {
			sink.Next(payload.NewString(fmt.Sprintf("Hello@%d", i), extension.TextPlain.String()))
		}
		sink.Complete()
	})

	// Or use Just.
	//f := flux.Just(
	//	payload.NewString("foo", extension.TextPlain.String()),
	//	payload.NewString("bar", extension.TextPlain.String()),
	//	payload.NewString("qux", extension.TextPlain.String()),
	//)

	// Block
	_, _ = f.
		DoOnNext(func(elem payload.Payload) error {
			// Handle and consume elements
			// Do something here...
			fmt.Println("bingo:", elem)
			return nil
		}).
		BlockLast(context.Background())

	// Subscribe
	f.Subscribe(context.Background(), rx.OnNext(func(input payload.Payload) error {
		fmt.Println("bingo:", input)
		return nil
	}))

	// Or implement your own subscriber
	var s rx.Subscriber
	f.SubscribeWith(context.Background(), s)
}

```

#### Backpressure & RequestN

`Flux` support **backpressure**.

You can call func `Request` in `Subscription` or use `LimitRate` before subscribe.

```go
package main

import (
	"context"
	"fmt"

	"github.com/rsocket/rsocket-go/extension"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
)

func main() {
	// Here is an example which consume Payload one by one.
	f := flux.Create(func(ctx context.Context, s flux.Sink) {
		for i := 0; i < 5; i++ {
			s.Next(payload.NewString(fmt.Sprintf("Hello@%d", i), extension.TextPlain.String()))
		}
		s.Complete()
	})

	var su rx.Subscription
	f.
		DoOnRequest(func(n int) {
			fmt.Printf("requesting next %d element......\n", n)
		}).
		Subscribe(
			context.Background(),
			rx.OnSubscribe(func(s rx.Subscription) {
				// Init Request 1 element.
				su = s
				su.Request(1)
			}),
			rx.OnNext(func(elem payload.Payload) error {
				// Consume element, do something...
				fmt.Println("bingo:", elem)
				// Request for next one manually.
				su.Request(1)
				return nil
			}),
		)
}
```

#### Dependencies
 - [reactor-go](https://github.com/jjeffcaii/reactor-go)
 - [testify](https://github.com/stretchr/testify)
 - [websocket](https://github.com/gorilla/websocket)
