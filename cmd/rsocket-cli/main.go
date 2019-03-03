package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/mkideal/cli"
	clix "github.com/mkideal/cli/ext"
	"github.com/rsocket/rsocket-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	errRequiredArgument = "Required arguments are missing: '%s'"
)

type opts struct {
	cli.Helper
	*zap.Logger

	Debug           bool          `cli:"d, debug" usage:"Debug Output"`
	ServerMode      bool          `cli:"s, server" usage:"Start server instead of client"`
	Keepalive       clix.Duration `cli:"k, keepalive" name:"duration" usage:"Keepalive period" dft:"20s"`
	AckTimeout      clix.Duration `cli:"ackTimeout" name:"duration" usage:"ACK timeout period" dft:"30s"`
	MissedAcks      int           `cli:"missed-acks" name:"count" usage:"Missed ACK" dft:"3"`
	MetadataFormat  string        `cli:"metadataFormat" name:"mimeType" usage:"Metadata Format [json|cbor|binary|text|mime-type]" dft:"json"`
	DataFormat      string        `cli:"dataFormat" name:"mimeType" usage:"Data Format [json|cbor|binary|text|mime-type]" dft:"binary"`
	Setup           string        `cli:"setup" name:"setup" usage:"String input or @path/to/file for setup metadata"`
	Input           []string      `cli:"i, input" name:"input" usage:"String input, '-' (STDIN) or @path/to/file"`
	Timeout         clix.Duration `cli:"timeout" name:"duration" usage:"Timeout period"`
	Operations      int           `cli:"o, ops" name:"operations" usage:"Operation Count" dft:"1"`
	RequestN        int           `cli:"r, requestn" name:"requests" usage:"Request N credits"`
	FireAndForget   bool          `cli:"fnf" usage:"Fire and Forget"`
	MetadataPush    bool          `cli:"metadataPush" usage:"Metadata Push"`
	RequestResponse bool          `cli:"request" usage:"Request Response"`
	Stream          bool          `cli:"stream" usage:"Request Stream"`
	Channel         bool          `cli:"channel" usage:"Request Channel"`
}

type SyncFunc func() error

func (opts *opts) configureLogging() (err error) {
	if opts.Debug {
		rsocket.SetLoggerLevel(rsocket.LogLevelDebug)

		opts.Logger, err = zap.NewDevelopment()
	} else {
		rsocket.SetLoggerLevel(rsocket.LogLevelInfo)

		opts.Logger, err = zap.NewProduction()
	}

	if err != nil {
		return
	}

	rsLogger := opts.Logger.Named("rsocket").WithOptions(zap.AddCaller(), zap.AddCallerSkip(2))

	rsocket.SetLoggerDebug(func(format string, args ...interface{}) {
		rsLogger.Debug(fmt.Sprintf(format, args...))
	})
	rsocket.SetLoggerInfo(func(format string, args ...interface{}) {
		rsLogger.Info(fmt.Sprintf(format, args...))
	})
	rsocket.SetLoggerWarn(func(format string, args ...interface{}) {
		rsLogger.Warn(fmt.Sprintf(format, args...))
	})
	rsocket.SetLoggerError(func(format string, args ...interface{}) {
		rsLogger.Error(fmt.Sprintf(format, args...))
	})

	return
}

func (opts *opts) createResponder(ctxt context.Context, logger *zap.Logger) (responder rsocket.RSocket) {
	return rsocket.NewAbstractSocket(
		rsocket.MetadataPush(func(payload rsocket.Payload) {
			logger.Debug("MetadataPush", zap.Object("payload", PayloadMarshaler{payload}))

			println(string(payload.Metadata()))
		}),
		rsocket.FireAndForget(func(payload rsocket.Payload) {
			logger.Debug("FireAndForget", zap.Object("payload", PayloadMarshaler{payload}))

			println(string(payload.Data()))
		}),
		rsocket.RequestResponse(func(payload rsocket.Payload) rsocket.Mono {
			logger.Debug("RequestResponse", zap.Object("payload", PayloadMarshaler{payload}))

			println(string(payload.Data()))

			return rsocket.JustMono(opts.singleInputPayload(ctxt))
		}),
		rsocket.RequestStream(func(payload rsocket.Payload) rsocket.Flux {
			logger.Debug("RequestStream", zap.Object("payload", PayloadMarshaler{payload}))

			println(string(payload.Data()))

			return opts.inputPublisher()
		}),
		rsocket.RequestChannel(func(publisher rsocket.Publisher) rsocket.Flux {
			logger.Debug("RequestChannel")

			if flux, ok := publisher.(rsocket.Flux); ok {
				flux.
					DoOnNext(func(ctx context.Context, payload rsocket.Payload) {
						println(string(payload.Data()))
					}).
					DoOnError(func(ctx context.Context, err error) {
						println("channel error:", err)
					})
			}

			publisher.Subscribe(ctxt, func(ctx context.Context, payload rsocket.Payload) {
				logger.Debug("RequestChannel", zap.Object("payload", PayloadMarshaler{payload}))
			})

			return opts.inputPublisher()
		}),
	)
}

func (opts *opts) buildServer(ctxt context.Context, logger *zap.Logger, uri *url.URL) (server rsocket.Start) {
	responder := opts.createResponder(ctxt, logger)

	return rsocket.Receive().Acceptor(func(setup rsocket.SetupPayload, socket rsocket.RSocket) rsocket.RSocket {
		logger.Debug("Accpet connection with setup payload", zap.Object("setup", SetupMarshaler{setup}))

		opts.
			runAllOperations(ctxt, socket).
			SubscribeOn(rsocket.ElasticScheduler()).
			Subscribe(ctxt, func(ctx context.Context, payload rsocket.Payload) {
				logger.Debug("received payload", zap.Object("payload", PayloadMarshaler{payload}))
			})

		return responder
	}).Transport(uri.Host)
}

func (opts *opts) buildClient(ctxt context.Context, logger *zap.Logger, uri *url.URL) (client rsocket.ClientStarter, err error) {
	responder := opts.createResponder(ctxt, logger)
	builder := rsocket.Connect().
		KeepAlive(opts.Keepalive.Duration, opts.AckTimeout.Duration, opts.MissedAcks).
		MetadataMimeType(standardMimeType(opts.MetadataFormat)).
		DataMimeType(standardMimeType(opts.DataFormat))

	var setup rsocket.Payload

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

func standardMimeType(fmt string) string {
	switch fmt {
	case "", "json":
		return "application/json"
	case "cbor":
		return "application/cbor"
	case "binary":
		return "application/binary"
	case "text":
		return "text/plain"
	default:
		return fmt
	}
}

func (opts *opts) parseSetupPayload() (setup rsocket.Payload, err error) {
	switch {
	case strings.HasPrefix(opts.Setup, "@"):
		var data []byte

		filename := opts.Setup[1:]
		data, err = ioutil.ReadFile(filename)

		if err != nil {
			return
		}

		setup = rsocket.NewPayload(data, nil)

	case opts.Setup == "":
		setup = rsocket.NewPayload(nil, nil)

	default:
		setup = rsocket.NewPayloadString(opts.Setup, "")
	}

	return
}

func (opts *opts) runAllOperations(ctxt context.Context, socket rsocket.RSocket) rsocket.Flux {
	return rsocket.NewFlux(func(ctx context.Context, emitter rsocket.Emitter) {
		for i := 0; i < opts.Operations; i++ {
			if opts.FireAndForget {
				payload := opts.singleInputPayload(ctxt)

				opts.Logger.Debug("FireAndForget", zap.Object("payload", PayloadMarshaler{payload}))

				socket.FireAndForget(payload)
			} else if opts.MetadataPush {
				payload := opts.singleInputPayload(ctxt)

				opts.Logger.Debug("MetadataPush", zap.Object("payload", PayloadMarshaler{payload}))

				socket.MetadataPush(payload)
			} else if opts.RequestResponse {
				payload := opts.singleInputPayload(ctxt)

				opts.Logger.Debug("RequestResponse", zap.Object("payload", PayloadMarshaler{payload}))

				socket.RequestResponse(payload)
			} else if opts.Stream {
				payload := opts.singleInputPayload(ctxt)

				opts.Logger.Debug("Stream", zap.Object("payload", PayloadMarshaler{payload}))

				socket.RequestStream(payload)
			} else if opts.Channel {
				opts.Logger.Debug("Channel")

				socket.RequestChannel(opts.inputPublisher())
			}
		}

		emitter.Complete()
	})
}

func (opts *opts) singleInputPayload(ctxt context.Context) rsocket.Payload {
	c := make(chan rsocket.Payload, 1)

	subscriber := opts.inputPublisher().Subscribe(ctxt, func(ctx context.Context, item rsocket.Payload) {
		c <- item
	})

	defer subscriber.Dispose()

	select {
	case payload := <-c:
		return payload

	case <-ctxt.Done():
		return nil
	}
}

func (opts *opts) inputPublisher() rsocket.Flux {
	return rsocket.NewFlux(func(ctx context.Context, emitter rsocket.Emitter) {
		defer emitter.Complete()

		for _, input := range opts.Input {
			input = strings.TrimSpace(input)

			switch {
			case input == "-":
				emitLines(ctx, emitter, os.Stdin)

			case strings.HasPrefix(input, "@"):
				filename := input[1:]
				f, err := os.Open(filename)

				if err != nil {
					opts.Logger.Warn("fail to open file", zap.String("filename", filename), zap.Error(err))

					emitter.Error(err)
				}

				defer f.Close()

				emitLines(ctx, emitter, os.Stdin)

			default:
				emitter.Next(rsocket.NewPayloadString(input, ""))
			}
		}
	})
}

func emitLines(ctxt context.Context, emitter rsocket.Emitter, r io.Reader) {
	ch := readLines(r)

	for {
		select {
		case <-ctxt.Done():
			break

		case line := <-ch:
			switch i := line.(type) {
			case error:
				emitter.Error(i)
			case string:
				emitter.Next(rsocket.NewPayloadString(i, ""))
			}
		}
	}
}

func readLines(r io.Reader) (ch chan interface{}) {
	ch = make(chan interface{}, 1)

	go func() {
		defer close(ch)

		br := bufio.NewReader(r)

		for {
			line, err := br.ReadString('\n')

			if err != nil {
				ch <- err
				break
			}

			ch <- line
		}
	}()

	return
}

type PayloadMarshaler struct {
	rsocket.Payload
}

func (payload PayloadMarshaler) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddByteString("data", payload.Data())
	encoder.AddByteString("metadata", payload.Metadata())

	return nil
}

type SetupMarshaler struct {
	rsocket.SetupPayload
}

func (setup SetupMarshaler) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	encoder.AddString("dataMimeType", setup.DataMimeType())
	encoder.AddString("metadataMimeType", setup.MetadataMimeType())
	encoder.AddDuration("timeBetweenKeepalive", setup.TimeBetweenKeepalive())
	encoder.AddDuration("maxLifetime", setup.MaxLifetime())
	encoder.AddString("version", setup.Version().String())
	encoder.AddObject("payload", PayloadMarshaler{setup})

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
			err = fmt.Errorf(errRequiredArgument, "target")
			return
		}

		logger.Debug("parsed opts", zap.Reflect("opts", opts))

		ctxt, cancel := context.WithCancel(context.Background())
		defer cancel()

		var uri *url.URL

		if uri, err = url.Parse(args[0]); err != nil {
			err = fmt.Errorf("Invalid URL, %v", err)
			return
		}

		if len(uri.Scheme) > 0 && !strings.HasPrefix(uri.Scheme, "tcp") {
			err = errors.New("Only support TCP.")
			return
		}

		if opts.ServerMode {
			serverLogger := logger.Named("server")
			defer serverLogger.Sync()

			server := opts.buildServer(ctxt, serverLogger, uri)

			logger.Info("start server", zap.Stringer("uri", uri))

			if err = server.Serve(); err != nil {
				logger.Error("server crashed", zap.Error(err))
			}
		}

		clientLogger := logger.Named("client")
		defer clientLogger.Sync()

		var client rsocket.ClientStarter

		if client, err = opts.buildClient(ctxt, clientLogger, uri); err != nil {
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
			ctxt, cancel = context.WithTimeout(ctxt, opts.Timeout.Duration)
		}

		subscripber := opts.
			runAllOperations(ctxt, socket).
			DoFinally(func(ctx context.Context) {
				cancel()
			}).
			SubscribeOn(rsocket.ElasticScheduler()).
			Subscribe(ctxt, func(ctx context.Context, payload rsocket.Payload) {
				logger.Debug("received payload", zap.Object("payload", PayloadMarshaler{payload}))
			})

		defer subscripber.Dispose()

		<-ctxt.Done()

		return
	}, "CLI for RSocket.")
}
