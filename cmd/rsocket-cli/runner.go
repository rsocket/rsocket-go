package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

type Runner struct {
	Stream         bool
	Request        bool
	FNF            bool
	Channel        bool
	MetadataPush   bool
	ServerMode     bool
	Input          string
	Metadata       string
	MetadataFormat string
	DataFormat     string
	Setup          string
	Debug          bool
	Ops            int
	Timeout        time.Duration
	Keepalive      time.Duration
	N              int
	Resume         bool
	URI            string
}

func (p *Runner) preflight() (err error) {
	if p.Debug {
		logger.SetLevel(logger.LevelDebug)
	}
	return
}

func (p *Runner) Run() error {
	if err := p.preflight(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if p.ServerMode {
		return p.runServerMode(ctx)
	}
	return p.runClientMode(ctx)
}

func (p *Runner) runClientMode(ctx context.Context) (err error) {
	cb := rsocket.Connect()
	if p.Resume {
		cb = cb.Resume()
	}
	setupData, err := p.readData(p.Setup)
	if err != nil {
		return
	}
	setupPayload := payload.New(setupData, nil)
	sendings := p.createPayload()
	c, err := cb.
		DataMimeType(p.DataFormat).
		MetadataMimeType(p.MetadataFormat).
		SetupPayload(setupPayload).
		Transport(p.URI).
		Start(ctx)
	if err != nil {
		return
	}
	defer func() {
		_ = c.Close()
	}()

	for i := 0; i < p.Ops; i++ {
		if i > 0 {
			logger.Infof("\n")
		}
		var first payload.Payload
		if !p.Channel {
			first, err = sendings.BlockFirst(ctx)
			if err != nil {
				return
			}
		}

		if p.Request {
			err = p.execRequestResponse(ctx, c, first)
		} else if p.FNF {
			err = p.execFireAndForget(ctx, c, first)
		} else if p.Stream {
			err = p.execRequestStream(ctx, c, first)
		} else if p.Channel {
			err = p.execRequestChannel(ctx, c, sendings)
		} else if p.MetadataPush {
			err = p.execMetadataPush(ctx, c, first)
		} else {
			err = p.execRequestResponse(ctx, c, first)
		}
		if err != nil {
			break
		}
	}
	return
}

func (p *Runner) runServerMode(ctx context.Context) error {
	var sb rsocket.ServerBuilder
	if p.Resume {
		sb = rsocket.Receive().Resume()
	} else {
		sb = rsocket.Receive()
	}
	ch := make(chan error)
	go func() {
		sendings := p.createPayload()
		var first payload.Payload
		if !p.Channel {
			var err error
			first, err = sendings.BlockFirst(ctx)
			if err != nil {
				ch <- err
				return
			}
		}
		ch <- sb.
			Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
				var options []rsocket.OptAbstractSocket
				if p.Channel {

				} else if p.Stream {

				} else {
					options = append(options, rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
						p.showPayload(msg)
						return mono.Just(first)
					}))
				}
				return rsocket.NewAbstractSocket(options...), nil
			}).
			Transport(p.URI).
			Serve(ctx)
		close(ch)
	}()
	return <-ch
}

func (p *Runner) execMetadataPush(ctx context.Context, c rsocket.Client, send payload.Payload) (err error) {
	c.MetadataPush(send)
	return
}

func (p *Runner) execFireAndForget(ctx context.Context, c rsocket.Client, send payload.Payload) (err error) {
	c.FireAndForget(send)
	return
}

func (p *Runner) execRequestResponse(ctx context.Context, c rsocket.Client, send payload.Payload) (err error) {
	res, err := c.RequestResponse(send).Block(ctx)
	if err != nil {
		return
	}
	p.showPayload(res)
	return
}

func (p *Runner) execRequestChannel(ctx context.Context, c rsocket.Client, send flux.Flux) error {
	var f flux.Flux
	if p.N < rx.RequestMax {
		f = c.RequestChannel(send).Take(p.N)
	} else {
		f = c.RequestChannel(send)
	}
	return p.printFlux(ctx, f)
}

func (p *Runner) execRequestStream(ctx context.Context, c rsocket.Client, send payload.Payload) error {
	var f flux.Flux
	if p.N < rx.RequestMax {
		f = c.RequestStream(send).Take(p.N)
	} else {
		f = c.RequestStream(send)
	}
	return p.printFlux(ctx, f)
}

func (p *Runner) printFlux(ctx context.Context, f flux.Flux) (err error) {
	var requested int
	_, err = f.
		DoOnNext(func(input payload.Payload) {
			if requested == 0 {
				p.showPayload(input)
			} else {
				logger.Infof("\n")
				p.showPayload(input)
			}
			requested++
		}).
		BlockLast(ctx)
	return
}

func (p *Runner) showPayload(pa payload.Payload) {
	logger.Infof("%s", pa.DataUTF8())
}

func (p *Runner) createPayload() flux.Flux {
	var md []byte
	if strings.HasPrefix(p.Metadata, "@") {
		var err error
		md, err = ioutil.ReadFile(p.Metadata[1:])
		if err != nil {
			return flux.Error(err)
		}
	} else {
		md = []byte(p.Metadata)
	}

	if p.Input == "-" {
		fmt.Println("Type commands to send to the server......")
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		return flux.Just(payload.New([]byte(strings.Trim(text, "\n")), md))
	}

	if !strings.HasPrefix(p.Input, "@") {
		return flux.Just(payload.New([]byte(p.Input), md))
	}

	return flux.Create(func(ctx context.Context, s flux.Sink) {
		f, err := os.Open(p.Input[1:])
		if err != nil {
			fmt.Println("error:", err)
			s.Error(err)
			return
		}
		defer func() {
			_ = f.Close()
		}()
		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				s.Error(ctx.Err())
				return
			default:
				s.Next(payload.New([]byte(scanner.Text()), md))
			}
		}
		s.Complete()
	})
}

func (p *Runner) readData(input string) (data []byte, err error) {
	switch {
	case strings.HasPrefix(input, "@"):
		data, err = ioutil.ReadFile(input[1:])
	case input != "":
		data = []byte(input)
	}
	return
}
