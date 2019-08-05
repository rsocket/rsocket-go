package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rsocket/rsocket-go/extension"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/rx"
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
	app.UsageText = "rsocket-cli [global options] [URI]"
	app.Name = "rsocket-cli"
	app.Usage = "CLI for RSocket."
	app.Version = "alpha"
	app.Flags = newFlags(conf)
	app.ArgsUsage = "[URI]"
	app.Action = func(c *cli.Context) (err error) {
		if c.NArg() < 1 {
			cli.ShowAppHelpAndExit(c, 1)
			return
		}
		conf.URI = c.Args().First()
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
			Value:       extension.ApplicationJSON.String(),
			Destination: &(args.MetadataFormat),
		},
		cli.StringFlag{
			Name:        "dataFormat",
			Usage:       "Data Format",
			Value:       "application/binary",
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
