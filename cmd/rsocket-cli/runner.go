package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rsocket/rsocket-go/internal/bytesconv"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/urfave/cli/v2"
)

var errConflictHeadersAndMetadata = errors.New("can't specify headers and metadata")

// Runner can be used to execute a command.
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
	wsHeaders        http.Header
}

func (r *Runner) preflight() (err error) {
	if r.Debug {
		logger.SetLevel(logger.LevelDebug)
	}

	headers := r.Headers.Value()

	if len(headers) > 0 && len(r.Metadata) > 0 {
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
		r.Metadata = string(bs)
	}
	if v := r.TransportHeaders.Value(); len(v) > 0 {
		r.wsHeaders = make(http.Header)
		for _, it := range v {
			idx := strings.Index(it, ":")
			if idx < 0 {
				return fmt.Errorf("invalid transport header: %s", it)
			}
			k := strings.TrimSpace(it[:idx])
			v := strings.TrimSpace(it[idx+1:])
			r.wsHeaders.Add(k, v)
		}
	}
	return
}

// Run run the Runner.
func (r *Runner) Run() error {
	if err := r.preflight(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if r.ServerMode {
		return r.runServerMode(ctx)
	}
	return r.runClientMode(ctx)
}

func (r *Runner) runClientMode(ctx context.Context) (err error) {
	cb := rsocket.Connect()
	if r.Resume {
		cb = cb.Resume()
	}
	setupData, err := r.readData(r.Setup)
	if err != nil {
		return
	}
	setupPayload := payload.New(setupData, nil)
	sendingPayloads := r.createPayload()

	tp, err := r.newClientTransport()
	if err != nil {
		return
	}
	c, err := cb.
		DataMimeType(r.DataFormat).
		MetadataMimeType(r.MetadataFormat).
		SetupPayload(setupPayload).
		Transport(tp).
		Start(ctx)
	if err != nil {
		return
	}
	defer func() {
		_ = c.Close()
	}()

	initialRequest := payload.NewString("", "")

	for i := 0; i < r.Ops; i++ {
		if i > 0 {
			logger.Infof("\n")
		}
		var first payload.Payload
		if !r.Channel {
			first, err = sendingPayloads.BlockFirst(ctx)
			if err != nil {
				return
			}
		}

		if r.Request {
			err = r.execRequestResponse(ctx, c, first)
		} else if r.FNF {
			err = r.execFireAndForget(ctx, c, first)
		} else if r.Stream {
			err = r.execRequestStream(ctx, c, first)
		} else if r.Channel {
			err = r.execRequestChannel(ctx, c, initialRequest, sendingPayloads)
		} else if r.MetadataPush {
			err = r.execMetadataPush(ctx, c, first)
		} else {
			err = r.execRequestResponse(ctx, c, first)
		}
		if err != nil {
			break
		}
	}
	return
}

func (r *Runner) runServerMode(ctx context.Context) error {
	var sb rsocket.ServerBuilder
	if r.Resume {
		sb = rsocket.Receive().Resume()
	} else {
		sb = rsocket.Receive()
	}
	ch := make(chan error)

	tp, err := r.newServerTransport()
	if err != nil {
		return err
	}

	go func() {
		sendingPayloads := r.createPayload()
		ch <- sb.
			Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
				var options []rsocket.OptAbstractSocket
				options = append(options, rsocket.RequestStream(func(message payload.Payload) flux.Flux {
					r.showPayload(message)
					return sendingPayloads
				}))
				options = append(options, rsocket.RequestChannel(func(initialRequest payload.Payload, messages flux.Flux) flux.Flux {
					messages.Subscribe(ctx, rx.OnNext(func(input payload.Payload) error {
						r.showPayload(input)
						return nil
					}))
					return sendingPayloads
				}))
				options = append(options, rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
					r.showPayload(msg)
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
					r.showPayload(msg)
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

func (r *Runner) execMetadataPush(_ context.Context, c rsocket.Client, send payload.Payload) (err error) {
	c.MetadataPush(send)
	m, _ := send.MetadataUTF8()
	logger.Infof("%s\n", m)
	return
}

func (r *Runner) execFireAndForget(_ context.Context, c rsocket.Client, send payload.Payload) (err error) {
	c.FireAndForget(send)
	return
}

func (r *Runner) execRequestResponse(ctx context.Context, c rsocket.Client, send payload.Payload) (err error) {
	res, release, err := c.RequestResponse(send).BlockUnsafe(ctx)
	if err != nil {
		return
	}
	defer release()
	r.showPayload(res)
	return
}

func (r *Runner) execRequestChannel(ctx context.Context, c rsocket.Client, initialRequest payload.Payload, send flux.Flux) error {
	var f flux.Flux
	if r.N < rx.RequestMax {
		f = c.RequestChannel(initialRequest, send).Take(r.N)
	} else {
		f = c.RequestChannel(initialRequest, send)
	}
	return r.printFlux(ctx, f)
}

func (r *Runner) execRequestStream(ctx context.Context, c rsocket.Client, send payload.Payload) error {
	var f flux.Flux
	if r.N < rx.RequestMax {
		f = c.RequestStream(send).Take(r.N)
	} else {
		f = c.RequestStream(send)
	}
	return r.printFlux(ctx, f)
}

func (r *Runner) printFlux(ctx context.Context, f flux.Flux) (err error) {
	_, err = f.
		DoOnNext(func(input payload.Payload) error {
			r.showPayload(input)
			return nil
		}).
		BlockLast(ctx)
	return
}

func (r *Runner) showPayload(pa payload.Payload) {
	logger.Infof("%s\n", pa.DataUTF8())
}

func (r *Runner) createPayload() flux.Flux {
	var md []byte
	if strings.HasPrefix(r.Metadata, "@") {
		var err error
		md, err = ioutil.ReadFile(r.Metadata[1:])
		if err != nil {
			return flux.Error(err)
		}
	} else {
		md = bytesconv.StringToBytes(r.Metadata)
	}

	if r.Input == "-" {
		fmt.Println("Type commands to send to the server......")
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		return flux.Just(payload.New(bytesconv.StringToBytes(strings.Trim(text, "\n")), md))
	}

	if !strings.HasPrefix(r.Input, "@") {
		return flux.Just(payload.New(bytesconv.StringToBytes(r.Input), md))
	}

	return flux.Create(func(ctx context.Context, s flux.Sink) {
		f, err := os.Open(r.Input[1:])
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
				s.Next(payload.New(bytesconv.StringToBytes(scanner.Text()), md))
			}
		}
		s.Complete()
	})
}

func (r *Runner) readData(input string) (data []byte, err error) {
	switch {
	case strings.HasPrefix(input, "@"):
		data, err = ioutil.ReadFile(input[1:])
	case input != "":
		data = bytesconv.StringToBytes(input)
	}
	return
}

func (r *Runner) newClientTransport() (transport.ClientTransporter, error) {
	u, err := url.Parse(r.URI)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "tcp":
		port, err := strconv.Atoi(u.Port())
		if err != nil {
			return nil, err
		}
		return rsocket.TCPClient().SetHostAndPort(u.Hostname(), port).Build(), nil
	case "unix":
		return rsocket.UnixClient().SetPath(u.Hostname()).Build(), nil
	case "ws", "wss":
		return rsocket.WebsocketClient().SetURL(r.URI).SetHeader(r.wsHeaders).Build(), nil
	default:
		return nil, fmt.Errorf("invalid transport %s", u.Scheme)
	}
}

func (r *Runner) newServerTransport() (transport.ServerTransporter, error) {
	u, err := url.Parse(r.URI)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "tcp":
		port, err := strconv.Atoi(u.Port())
		if err != nil {
			return nil, err
		}
		return rsocket.TCPServer().SetHostAndPort(u.Hostname(), port).Build(), nil
	case "unix":
		return rsocket.UnixServer().SetPath(u.Hostname()).Build(), nil
	case "ws", "wss":
		var addr string
		if port := u.Port(); port != "" {
			addr = fmt.Sprintf("%s:%s", u.Hostname(), port)
		} else {
			addr = fmt.Sprintf("%s:%d", u.Hostname(), rsocket.DefaultPort)
		}
		return rsocket.WebsocketServer().SetAddr(addr).SetPath(u.EscapedPath()).Build(), nil
	default:
		return nil, errors.Errorf("invalid transport %s", u.Scheme)
	}
}
