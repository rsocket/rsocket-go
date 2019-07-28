package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/urfave/cli"
)

func init() {
	logger.DisablePrefix()
	fn := func(s string, i ...interface{}) {
		fmt.Printf(s, i...)
	}
	logger.SetFunc(logger.LevelDebug, fn)
	logger.SetFunc(logger.LevelInfo, fn)
	logger.SetFunc(logger.LevelError, func(s string, i ...interface{}) {
		_, _ = os.Stderr.WriteString(fmt.Sprintf(s, i...))
	})
}

func main() {
	conf := &Runner{}
	app := cli.NewApp()
	//app.UsageText = "rsocket-cli [global options] [target]"
	app.Name = "rsocket-cli"
	app.Usage = "CLI for RSocket."
	app.Version = "alpha"
	app.Flags = newFlags(conf)
	app.ArgsUsage = "[target]"
	app.Action = func(c *cli.Context) (err error) {
		conf.URI = c.Args().Get(0)
		return conf.Run()
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func newFlags(args *Runner) []cli.Flag {
	return []cli.Flag{
		cli.StringSliceFlag{
			Name:  "header,H",
			Usage: "Custom header to pass to server",
		},
		cli.StringSliceFlag{
			Name:  "transport-header,T",
			Usage: "Custom header to pass to the transport",
		},
		cli.BoolFlag{
			Name:        "stream",
			Usage:       "Request Stream",
			Destination: &(args.Stream),
		},
		cli.BoolFlag{
			Name:        "request",
			Usage:       "Request Response",
			Destination: &(args.Request),
		},
		cli.BoolFlag{
			Name:        "fnf",
			Usage:       "Fire And Forget",
			Destination: &(args.FNF),
		},
		cli.BoolFlag{
			Name:        "channel",
			Usage:       "Channel",
			Destination: &(args.Channel),
		},
		cli.BoolFlag{
			Name:        "metadataPush",
			Usage:       "Metadata Push",
			Destination: &(args.MetadataPush),
		},
		cli.BoolFlag{
			Name:        "server,s",
			Usage:       "Start server instead of client",
			Destination: &(args.ServerMode),
		},
		cli.StringFlag{
			Name:        "input,i",
			Usage:       "String input, '-' (STDIN) or @path/to/file",
			Destination: &(args.Input),
		},
		cli.StringFlag{
			Name:        "metadata, m",
			Usage:       "Metadata input string input or @path/to/file",
			Destination: &(args.Metadata),
		},
		cli.StringFlag{
			Name:        "metadataFormat",
			Usage:       "Metadata Format",
			Value:       "json",
			Destination: &(args.MetadataFormat),
		},
		cli.StringFlag{
			Name:        "dataFormat",
			Usage:       "Data Format",
			Value:       "binary",
			Destination: &(args.DataFormat),
		},
		cli.StringFlag{
			Name:        "setup",
			Usage:       "String input or @path/to/file for setup metadata",
			Destination: &(args.Setup),
		},
		cli.BoolFlag{
			Name:        "debug,d",
			Usage:       "Debug Output",
			Destination: &(args.Debug),
		},
		cli.IntFlag{
			Name:        "ops",
			Usage:       "Operation Count",
			Value:       1,
			Destination: &(args.Ops),
		},
		cli.DurationFlag{
			Name:        "timeout",
			Usage:       "Timeout in seconds",
			Destination: &(args.Timeout),
		},
		cli.DurationFlag{
			Name:        "keepalive,k",
			Usage:       "Keepalive period",
			Value:       20 * time.Second,
			Destination: &(args.Keepalive),
		},
		cli.IntFlag{
			Name:        "requestn, r",
			Usage:       "Request N credits",
			Value:       rx.RequestMax,
			Destination: &(args.N),
		},
		cli.BoolFlag{
			Name:        "resume",
			Usage:       "resume enabled",
			Destination: &(args.Resume),
		},
	}

}

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

func (p *Runner) PreFlight() (err error) {
	if p.Debug {
		logger.SetLevel(logger.LevelDebug)
	}
	if p.Input == "-" {
		fmt.Println("Type commands to send to the server......")
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		p.Input = strings.Trim(text, "\n")
	}
	return
}

func (p *Runner) Run() error {
	if err := p.PreFlight(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		cancel()
	}()

	go func() {
		cs := make(chan os.Signal, 1)
		signal.Notify(cs, os.Interrupt, syscall.SIGTERM)
		<-cs
		cancel()
	}()

	if p.ServerMode {
		return p.runServerMode(ctx)
	}
	return p.runClientMode(ctx)
}

func (p *Runner) runClientMode(ctx context.Context) (err error) {
	var cb rsocket.ClientBuilder
	if p.Resume {
		cb = rsocket.Connect().Resume()
	} else {
		cb = rsocket.Connect()
	}
	c, err := cb.
		DataMimeType(p.DataFormat).
		MetadataMimeType(p.MetadataFormat).
		SetupPayload(payload.NewString(p.Setup, "")).
		Transport(p.URI).
		Start(context.Background())
	if err != nil {
		return
	}
	defer func() {
		_ = c.Close()
	}()
	send := p.CreatePayload()
	if p.Stream {
		err = p.RequestStream(ctx, c, send)
	} else {
		err = p.RequestResponse(ctx, c, send)
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
		err := sb.
			Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) rsocket.RSocket {
				var options []rsocket.OptAbstractSocket

				if p.Channel {

				} else if p.Stream {

				} else {
					options = append(options, rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
						p.showPayload(msg)
						return mono.Just(p.CreatePayload())
					}))
				}
				return rsocket.NewAbstractSocket(options...)
			}).
			Transport(p.URI).
			Serve(ctx)
		ch <- err
		close(ch)
	}()
	return <-ch
}

func (p *Runner) RequestResponse(ctx context.Context, c rsocket.Client, send payload.Payload) (err error) {
	res, err := c.RequestResponse(send).Block(ctx)
	if err != nil {
		return
	}
	p.showPayload(res)
	return
}

func (p *Runner) RequestStream(ctx context.Context, c rsocket.Client, send payload.Payload) (err error) {
	first := true
	_, err = c.RequestStream(send).
		DoOnNext(func(input payload.Payload) {
			if first {
				first = false
				p.showPayload(input)
			} else {
				logger.Infof("\n")
				p.showPayload(input)
			}
		}).
		BlockLast(ctx)
	return
}

func (p *Runner) showPayload(pa payload.Payload) {
	logger.Infof("%s", pa.DataUTF8())
}

func (p *Runner) CreatePayload() payload.Payload {
	return payload.NewString(p.Input, p.Metadata)
}
