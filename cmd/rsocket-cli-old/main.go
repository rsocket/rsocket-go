package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"

	"github.com/2tvenom/cbor"
	"github.com/mkideal/cli"
	clix "github.com/mkideal/cli/ext"
	"github.com/rsocket/rsocket-go"
	rlog "github.com/rsocket/rsocket-go/common/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	appJSON   = "application/json"
	appCBOR   = "application/cbor"
	appBinary = "application/binary"
	textPlain = "text/plain"
)

type opts struct {
	cli.Helper
	*zap.Logger

	Debug           bool              `cli:"d, debug" usage:"Debug Output"`
	ServerMode      bool              `cli:"s, server" usage:"Start server instead of client"`
	Keepalive       clix.Duration     `cli:"k, keepalive" name:"duration" usage:"Keepalive period" dft:"20s"`
	AckTimeout      clix.Duration     `cli:"ackTimeout" name:"duration" usage:"ACK timeout period" dft:"30s"`
	MissedAcks      int               `cli:"missed-acks" name:"count" usage:"Missed ACK" dft:"3"`
	MetadataFmt     string            `cli:"metadataFormat" name:"mimeType" usage:"Metadata Format [json|cbor|binary|text|mime-type]" dft:"json"`
	DataFmt         string            `cli:"dataFormat" name:"mimeType" usage:"Data Format [json|cbor|binary|text|mime-type]" dft:"binary"`
	Setup           string            `cli:"setup" name:"setup" usage:"String input or @path/to/file for setup metadata"`
	Metadata        string            `cli:"m, metadata" name:"metadata" usage:"Metadata input string input or @path/to/file"`
	Headers         map[string]string `cli:"H, header" name:"name=value" usage:"Request Response"`
	Input           []string          `cli:"i, input" name:"input" usage:"String input, '-' (STDIN) or @path/to/file"`
	Timeout         clix.Duration     `cli:"timeout" name:"duration" usage:"Timeout period"`
	Operations      int               `cli:"o, ops" name:"operations" usage:"Operation Count" dft:"1"`
	RequestN        int               `cli:"r, requestn" name:"requests" usage:"Request N credits" dft:"-1"`
	FireAndForget   bool              `cli:"fnf" usage:"Fire and Forget"`
	MetadataPush    bool              `cli:"metadataPush" usage:"Metadata Push"`
	RequestResponse bool              `cli:"request" usage:"Request Response"`
	Stream          bool              `cli:"stream" usage:"Request Stream"`
	Channel         bool              `cli:"channel" usage:"Request Channel"`
}

func (opts *opts) configureLogging() (err error) {
	if opts.Debug {
		rlog.SetLoggerLevel(rlog.LogLevelDebug)

		opts.Logger, err = zap.NewDevelopment()
	} else {
		rlog.SetLoggerLevel(rlog.LogLevelInfo)

		opts.Logger, err = zap.NewProduction()
	}

	if err != nil {
		return
	}

	rsLogger := opts.Logger.Named("rsocket").WithOptions(zap.AddCaller(), zap.AddCallerSkip(2))

	rlog.SetLoggerDebug(func(format string, args ...interface{}) {
		rsLogger.Debug(fmt.Sprintf(format, args...))
	})
	rlog.SetLoggerInfo(func(format string, args ...interface{}) {
		rsLogger.Info(fmt.Sprintf(format, args...))
	})
	rlog.SetLoggerWarn(func(format string, args ...interface{}) {
		rsLogger.Warn(fmt.Sprintf(format, args...))
	})
	rlog.SetLoggerError(func(format string, args ...interface{}) {
		rsLogger.Error(fmt.Sprintf(format, args...))
	})

	return
}

func (opts *opts) createResponder(ctx context.Context, logger *zap.Logger) (responder rsocket.RSocket) {
	return rsocket.NewAbstractSocket(
		rsocket.MetadataPush(func(payload payload.Payload) {
			logger.Debug("MetadataPush", zap.Object("request", payloadMarshaler{payload}))
			metadata, _ := payload.MetadataUTF8()
			println(metadata)
		}),
		rsocket.FireAndForget(func(payload payload.Payload) {
			logger.Debug("FireAndForget", zap.Object("request", payloadMarshaler{payload}))

			println(payload.DataUTF8())
		}),
		rsocket.RequestResponse(func(payload payload.Payload) rx.Mono {
			logger.Debug("RequestResponse", zap.Object("request", payloadMarshaler{payload}))

			println(payload.DataUTF8())

			return rx.JustMono(opts.singleInputPayload(ctx))
		}),
		rsocket.RequestStream(func(payload payload.Payload) rx.Flux {
			logger.Debug("RequestStream", zap.Object("request", payloadMarshaler{payload}))

			println(payload.DataUTF8())

			return opts.inputPublisher()
		}),
		rsocket.RequestChannel(func(publisher rx.Publisher) rx.Flux {
			logger.Debug("RequestChannel")
			rx.ToFlux(publisher).
				DoOnNext(func(ctx context.Context, s rx.Subscription, payload payload.Payload) {
					println(payload.DataUTF8())
				}).
				DoOnError(func(ctx context.Context, err error) {
					println("channel error:", err)
				}).
				Subscribe(ctx, rx.OnNext(func(ctx context.Context, s rx.Subscription, payload payload.Payload) {
					logger.Debug("RequestChannel", zap.Object("payload", payloadMarshaler{payload}))
				}))

			return opts.inputPublisher()
		}),
	)
}

func (opts *opts) buildServer(ctxt context.Context, logger *zap.Logger, uri *url.URL) (server rsocket.Start) {
	responder := opts.createResponder(ctxt, logger)
	return rsocket.Receive().Acceptor(func(setup payload.SetupPayload, socket rsocket.EnhancedRSocket) rsocket.RSocket {
		logger.Debug("Accpet connection with setup payload", zap.Object("setup", setupMarshaler{setup}))

		go opts.runAllOperations(ctxt, socket)

		return responder
	}).Transport(uri.Host)
}

func (opts *opts) buildClient(ctxt context.Context, logger *zap.Logger, uri *url.URL) (client rsocket.ClientStarter, err error) {
	responder := opts.createResponder(ctxt, logger)
	builder := rsocket.Connect().
		KeepAlive(opts.Keepalive.Duration, opts.AckTimeout.Duration, opts.MissedAcks).
		MetadataMimeType(opts.MetadataFormat()).
		DataMimeType(opts.DataFormat())

	var setup payload.Payload

	if setup, err = opts.parseSetupPayload(); err != nil {
		logger.Error("fail to parse setup payload", zap.Error(err))

		return
	}

	if setup != nil {
		builder.SetupPayload(setup)
	}

	client = builder.Acceptor(func(socket rsocket.RSocket) rsocket.RSocket {
		logger.Debug("client connected")

		return responder
	}).Transport(uri.Host)

	return
}

func (opts *opts) MetadataFormat() string {
	return standardMimeType(opts.MetadataFmt)
}

func (opts *opts) DataFormat() string {
	return standardMimeType(opts.DataFmt)
}

func standardMimeType(fmt string) string {
	switch fmt {
	case "", "json":
		return appJSON
	case "cbor":
		return appCBOR
	case "binary":
		return appBinary
	case "text":
		return textPlain
	default:
		return fmt
	}
}

func readData(s string) (data []byte, err error) {
	switch {
	case strings.HasPrefix(s, "@"):
		return ioutil.ReadFile(s[1:])

	case s == "":
		break

	default:
		data = []byte(s)
	}

	return
}

func (opts *opts) parseSetupPayload() (setup payload.Payload, err error) {
	var data []byte

	data, err = readData(opts.Setup)

	if err != nil {
		return
	}

	setup = payload.New(data, nil)

	return
}

func (opts *opts) buildMetadata() (metadata []byte, err error) {
	if len(opts.Metadata) > 0 && len(opts.Headers) > 0 {
		err = errors.New("can't specify headers and metadata")
	} else if len(opts.Metadata) > 0 {
		metadata, err = readData(opts.Metadata)
	} else if len(opts.Headers) > 0 {
		switch opts.MetadataFormat() {
		case appJSON:
			metadata, err = json.Marshal(opts.Headers)

		case appCBOR:
			var buf bytes.Buffer

			encoder := cbor.NewEncoder(&buf)

			_, err = encoder.Marshal(opts.Headers)

			metadata = buf.Bytes()

		default:
			err = fmt.Errorf("headers not supported with mimetype: %s", opts.MetadataFmt)
		}
	}

	return
}

func (opts *opts) runAllOperations(ctxt context.Context, socket rsocket.RSocket) {
	var wg sync.WaitGroup

	for i := 0; i < opts.Operations; i++ {
		if opts.FireAndForget {
			pl := opts.singleInputPayload(ctxt)

			opts.Logger.Debug("FireAndForget", zap.Object("request", payloadMarshaler{pl}))

			socket.FireAndForget(pl)
		} else if opts.MetadataPush {
			pl := opts.singleInputPayload(ctxt)

			opts.Logger.Debug("MetadataPush", zap.Object("request", payloadMarshaler{pl}))

			socket.MetadataPush(pl)
		} else if opts.RequestResponse {
			pl := opts.singleInputPayload(ctxt)

			opts.Logger.Debug("RequestResponse", zap.Object("request", payloadMarshaler{pl}))

			socket.
				RequestResponse(pl).
				SubscribeOn(rx.ElasticScheduler()).
				Subscribe(ctxt, rx.OnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
					defer wg.Done()

					opts.Logger.Debug("RequestResponse", zap.Object("response", payloadMarshaler{elem}))

					println(elem.DataUTF8())
				}))

			wg.Add(1)
		} else if opts.Stream {
			pl := opts.singleInputPayload(ctxt)

			opts.Logger.Debug("Stream", zap.Object("request", payloadMarshaler{pl}))

			var subscription rx.Disposable
			var responses int

			subscription = socket.
				RequestStream(pl).
				DoFinally(func(ctx context.Context, sig rx.SignalType) {
					wg.Done()
				}).
				SubscribeOn(rx.ElasticScheduler()).
				Subscribe(ctxt, rx.OnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
					opts.Logger.Debug("Stream", zap.Object("response", payloadMarshaler{elem}))

					println(elem.DataUTF8())

					responses++

					if opts.RequestN > 0 && responses >= opts.RequestN {
						subscription.Dispose()
					}
				}))

			wg.Add(1)
		} else if opts.Channel {
			opts.Logger.Debug("Channel")

			var responses int

			socket.
				RequestChannel(opts.inputPublisher()).
				DoFinally(func(ctx context.Context, sig rx.SignalType) {
					wg.Done()
				}).
				SubscribeOn(rx.ElasticScheduler()).
				Subscribe(ctxt, rx.OnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
					opts.Logger.Debug("Channel", zap.Object("response", payloadMarshaler{elem}))

					println(string(elem.Data()))

					responses++

					if opts.RequestN > 0 && responses >= opts.RequestN {
						s.Cancel()
					}
				}))

			wg.Add(1)
		}
	}

	wg.Wait()
}

func (opts *opts) singleInputPayload(ctxt context.Context) payload.Payload {
	c := make(chan payload.Payload, 1)

	subscriber := opts.inputPublisher().Subscribe(ctxt, rx.OnNext(func(ctx context.Context, s rx.Subscription, item payload.Payload) {
		c <- item
	}))

	defer subscriber.Dispose()

	select {
	case pl := <-c:
		return pl

	case <-ctxt.Done():
		return nil
	}
}

func (opts *opts) inputPublisher() rx.Flux {
	return rx.NewFlux(func(ctx context.Context, emitter rx.Producer) {
		defer emitter.Complete()

		metadata, err := opts.buildMetadata()

		if err != nil {
			opts.Logger.Warn("fail to read metadata", zap.Error(err))
		}

		for _, input := range opts.Input {
			switch {
			case input == "-":
				opts.Logger.Debug("emit lines from STDIN")

				publisher := lineInputPublishers{ctx, opts.Logger, os.Stdin}

				publisher.emitLines(emitter, string(metadata))

			case strings.HasPrefix(input, "@"):
				filename := input[1:]
				f, err := os.Open(filename)

				if err != nil {
					opts.Logger.Warn("fail to open file", zap.String("filename", filename), zap.Error(err))

					emitter.Error(err)
				}

				defer f.Close()

				opts.Logger.Debug("emit lines from file", zap.String("filename", filename))

				publisher := lineInputPublishers{ctx, opts.Logger, f}

				publisher.emitLines(emitter, string(metadata))

			default:
				_ = emitter.Next(payload.NewString(input, string(metadata)))
			}
		}
	})
}

type lineInputPublishers struct {
	context.Context
	*zap.Logger
	io.Reader
}

func (p *lineInputPublishers) emitLines(emitter rx.Producer, metadata string) {
	ch := make(chan interface{}, 1)

	go func() {
		defer close(ch)

		br := bufio.NewReader(p.Reader)

		for {
			line, err := br.ReadString('\n')

			if err != nil {
				ch <- err
				break
			}

			ch <- line
		}
	}()

	for {
		select {
		case <-p.Context.Done():
			return
		case line := <-ch:
			switch i := line.(type) {
			case error:
				emitter.Error(i)
			case string:
				_ = emitter.Next(payload.NewString(i, metadata))
			}
		}
	}
}

type payloadMarshaler struct {
	payload.Payload
}

func (payload payloadMarshaler) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddByteString("data", payload.Data())
	metadata, _ := payload.Metadata()
	encoder.AddByteString("metadata", metadata)

	return nil
}

type setupMarshaler struct {
	payload.SetupPayload
}

func (setup setupMarshaler) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("dataMimeType", setup.DataMimeType())
	encoder.AddString("metadataMimeType", setup.MetadataMimeType())
	encoder.AddDuration("timeBetweenKeepalive", setup.TimeBetweenKeepalive())
	encoder.AddDuration("maxLifetime", setup.MaxLifetime())
	encoder.AddString("version", setup.Version().String())
	_ = encoder.AddObject("payload", payloadMarshaler{setup})

	return nil
}

func main() {
	cli.Run(new(opts), func(cmdline *cli.Context) (err error) {
		opts := cmdline.Argv().(*opts)

		if err = opts.configureLogging(); err != nil {
			return
		}

		logger := opts.Logger

		defer logger.Sync()

		args := cmdline.Args()

		if len(args) == 0 {
			err = fmt.Errorf("required arguments are missing: '%s'", "target")
			return
		}

		logger.Debug("parsed opts", zap.Reflect("opts", opts))

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var uri *url.URL

		if uri, err = url.Parse(args[0]); err != nil {
			err = fmt.Errorf("invalid URL, %v", err)
			return
		}

		switch uri.Scheme {
		case "", "tcp", "tcp4", "tcp6":
			break
		default:
			err = errors.New("only support TCP")
			return
		}

		if opts.ServerMode {
			serverLogger := logger.Named("server")
			defer serverLogger.Sync()

			server := opts.buildServer(ctx, serverLogger, uri)

			logger.Info("start server", zap.Stringer("uri", uri))

			if err = server.Serve(); err != nil {
				logger.Error("server crashed", zap.Error(err))
			}
		}

		clientLogger := logger.Named("client")
		defer clientLogger.Sync()

		var client rsocket.ClientStarter

		if client, err = opts.buildClient(ctx, clientLogger, uri); err != nil {
			logger.Warn("fail to build client", zap.Stringer("uri", uri), zap.Error(err))

			return
		}

		logger.Info("start client", zap.Stringer("uri", uri))

		var socket rsocket.ClientSocket

		if socket, err = client.Start(); err != nil {
			logger.Warn("fail to start client", zap.Stringer("uri", uri), zap.Error(err))

			return
		}

		defer socket.Close()

		if opts.Timeout.Duration > 0 {
			ctx, cancel = context.WithTimeout(ctx, opts.Timeout.Duration)

			defer cancel()
		}

		c := make(chan struct{}, 1)

		go func() {
			opts.runAllOperations(ctx, socket)

			c <- struct{}{}

			close(c)
		}()

		select {
		case <-c:
			break

		case <-ctx.Done():
			break
		}

		return
	}, "CLI for RSocket.")
}
