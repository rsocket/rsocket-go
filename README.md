# rsocket-go
![logo](./logo.jpg)

![GitHub Workflow Status](https://github.com/rsocket/rsocket-go/workflows/Go/badge.svg)
[![codecov](https://codecov.io/gh/rsocket/rsocket-go/branch/master/graph/badge.svg)](https://codecov.io/gh/rsocket/rsocket-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/rsocket/rsocket-go)](https://goreportcard.com/report/github.com/rsocket/rsocket-go)
[![GoDoc](https://godoc.org/github.com/rsocket/rsocket-go?status.svg)](https://godoc.org/github.com/rsocket/rsocket-go)
![License](https://img.shields.io/github/license/rsocket/rsocket-go.svg)
[![GitHub Release](https://img.shields.io/github/release-pre/rsocket/rsocket-go.svg)](https://github.com/rsocket/rsocket-go/releases)

> rsocket-go is an implementation of the [RSocket](http://rsocket.io/) protocol in Go.

## Features
 - Design For Golang.
 - Thin [reactive-streams](http://www.reactive-streams.org/) implementation.
 - Simulate Java SDK API.
 - Fast CLI (Compatible with [https://github.com/rsocket/rsocket-cli](https://github.com/rsocket/rsocket-cli/)).
   - Installation: `go get github.com/rsocket/rsocket-go/cmd/rsocket-cli`
   - Example: `rsocket-cli --request -i hello_world --setup setup_me tcp://127.0.0.1:7878`

## Install

> Minimal go version is ***1.11***.

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
		Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			// bind responder
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
					return mono.Just(msg)
				}),
			), nil
		}).
		Transport(rsocket.TCPServer().SetAddr(":7878").Build()).
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
		Transport(rsocket.TCPClient().SetHostAndPort("127.0.0.1", 7878).Build()).
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

*rsocket-go* provides TCP/Websocket transport implementations by default. Since `v0.6.0`, you can use `core` package to implement your own RSocket transport.
I created an example project which show how to implement an unofficial [QUIC](https://en.wikipedia.org/wiki/QUIC) transport.
You can see [rsocket-transport-quic](https://github.com/jjeffcaii/rsocket-transport-quic) if you are interested.

## TODO

- [ ] Wiki
- [ ] UT: 90% coverage
