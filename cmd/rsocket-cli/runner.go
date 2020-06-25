package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/urfave/cli/v2"
)

var errConflictHeadersAndMetadata = errors.New("can't specify headers and metadata")

type Runner struct {
	Headers          cli.StringSlice
	TransportHeaders cli.StringSlice
	Stream           bool
	Request          bool
	FNF              bool
	Channel          bool
	MetadataPush     bool
	ServerMode       bool
	Input            string
	Metadata         string
	MetadataFormat   string
	DataFormat       string
	Setup            string
	Debug            bool
	Ops              int
	Timeout          time.Duration
	Keepalive        time.Duration
	N                int
	Resume           bool
	URI              string
	wsHeaders        map[string][]string
}

func (p *Runner) preflight() (err error) {
	if p.Debug {
		logger.SetLevel(logger.LevelDebug)
	}

	headers := p.Headers.Value()

	if len(headers) > 0 && len(p.Metadata) > 0 {
		return errConflictHeadersAndMetadata
	}
	if len(headers) > 0 {
		headers := make(map[string]string)
		for _, it := range headers {
			idx := strings.Index(it, ":")
			if idx < 0 {
				return fmt.Errorf("invalid header: %s", it)
			}
			k := it[:idx]
			v := it[idx+1:]
			headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
		bs, _ := json.Marshal(headers)
		p.Metadata = string(bs)
	}

	tpHeaders := p.TransportHeaders.Value()
	if len(tpHeaders) > 0 {
		headers := make(map[string][]string)
		for _, it := range tpHeaders {
			idx := strings.Index(it, ":")
			if idx < 0 {
				return fmt.Errorf("invalid transport header: %s", it)
			}
			k := strings.TrimSpace(it[:idx])
			v := strings.TrimSpace(it[idx+1:])
			headers[k] = append(headers[k], v)
		}
		p.wsHeaders = headers
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
	sendingPayloads := p.createPayload()

	tp, err := makeTransport(p.URI)
	if err != nil {
		return
	}

	// TODO:

	//if ws, ok := tp.(*rsocket.wsTransporter); ok {
	//	ws.Header(p.wsHeaders)
	//}

	c, err := cb.
		DataMimeType(p.DataFormat).
		MetadataMimeType(p.MetadataFormat).
		SetupPayload(setupPayload).
		Transport(tp).
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
			first, err = sendingPayloads.BlockFirst(ctx)
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
			err = p.execRequestChannel(ctx, c, sendingPayloads)
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

	tp, err := makeTransport(p.URI)
	if err != nil {
		return err
	}

	go func() {
		sendingPayloads := p.createPayload()
		ch <- sb.
			Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
				var options []rsocket.OptAbstractSocket
				options = append(options, rsocket.RequestStream(func(message payload.Payload) flux.Flux {
					p.showPayload(message)
					return sendingPayloads
				}))
				options = append(options, rsocket.RequestChannel(func(messages rx.Publisher) flux.Flux {
					messages.Subscribe(ctx, rx.OnNext(func(input payload.Payload) {
						p.showPayload(input)
					}))
					return sendingPayloads
				}))
				options = append(options, rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
					p.showPayload(msg)
					return mono.Create(func(i context.Context, sink mono.Sink) {
						first, err := sendingPayloads.BlockFirst(i)
						if err != nil {
							sink.Error(err)
							return
						}
						sink.Success(first)
					})
				}))
				options = append(options, rsocket.FireAndForget(func(msg payload.Payload) {
					p.showPayload(msg)
				}))
				options = append(options, rsocket.MetadataPush(func(msg payload.Payload) {
					metadata, _ := msg.MetadataUTF8()
					logger.Infof("%s\n", metadata)
				}))
				return rsocket.NewAbstractSocket(options...), nil
			}).
			Transport(tp).
			Serve(ctx)
		close(ch)
	}()
	return <-ch
}

func (p *Runner) execMetadataPush(_ context.Context, c rsocket.Client, send payload.Payload) (err error) {
	c.MetadataPush(send)
	m, _ := send.MetadataUTF8()
	logger.Infof("%s\n", m)
	return
}

func (p *Runner) execFireAndForget(_ context.Context, c rsocket.Client, send payload.Payload) (err error) {
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
	_, err = f.
		DoOnNext(func(input payload.Payload) {
			p.showPayload(input)
		}).
		BlockLast(ctx)
	return
}

func (p *Runner) showPayload(pa payload.Payload) {
	logger.Infof("%s\n", pa.DataUTF8())
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

func makeTransport(s string) (rsocket.Transporter, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "tcp":
		port, err := strconv.Atoi(u.Port())
		if err != nil {
			return nil, err
		}
		return rsocket.Tcp().HostAndPort(u.Hostname(), port).Build(), nil
	case "unix":
		return rsocket.Unix().Path(u.Hostname()).Build(), nil
	case "ws", "wss":
		return rsocket.Websocket().Url(s).Build(), nil
	default:
		return nil, fmt.Errorf("invalid transport %s", u.Scheme)
	}

}
