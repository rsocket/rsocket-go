package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/extension"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
)

const number = 13

func main() {

	readyCh := make(chan struct{})

	// start a server in a go routine
	go server(readyCh)

	// wait for the server to be ready
	<-readyCh

	// call the client
	client()
}

func server(readyCh chan struct{}) {
	// create a handler that will be called when the server receives the RequestChannel frame (FrameTypeRequestChannel - 0x07)
	requestChannelHandler := rsocket.RequestChannel(func(initialRequest payload.Payload, requests flux.Flux) flux.Flux {
		return flux.Create(func(ctx context.Context, s flux.Sink) {
			requests.DoOnNext(func(elem payload.Payload) error {
				// for each payload in a flux stream respond with a word count
				s.Next(payload.NewString(fmt.Sprintf("%d", wordCount(elem.DataUTF8())), ""))
				return nil
			}).DoOnComplete(func() {
				// signal completion of the response stream
				s.Complete()
			}).Subscribe(context.Background())
		})
	})

	err := rsocket.Receive().
		OnStart(func() {
			// close the channel to signal that the server is ready
			close(readyCh)
		}).
		Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			// register a new request channel handler
			return rsocket.NewAbstractSocket(requestChannelHandler), nil
		}).
		// specify transport
		Transport(rsocket.TCPServer().SetAddr(":7878").Build()).
		// serve will block execution unless an error occurred
		Serve(context.Background())

	panic(err)
}

func client() {
	// Start a client connection
	client, err := rsocket.Connect().Transport(rsocket.TCPClient().SetHostAndPort("127.0.0.1", 7878).Build()).Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// strings to count the words
	initialRequest := payload.NewString("", "")
	sentences := []payload.Payload{
		payload.NewString("", extension.TextPlain.String()),
		payload.NewString("qux", extension.TextPlain.String()),
		payload.NewString("The quick brown fox jumps over the lazy dog", extension.TextPlain.String()),
		payload.NewString("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.", extension.TextPlain.String()),
	}

	f := flux.FromSlice(sentences)

	// create a wait group so that the function does not return until the stream completes
	wg := sync.WaitGroup{}
	wg.Add(1)

	counter := 0

	// register handler for RequestChannel
	client.RequestChannel(initialRequest, f).DoOnNext(func(input payload.Payload) error {
		// print word count
		fmt.Println(sentences[counter].DataUTF8(), ":", input.DataUTF8())
		counter = counter + 1
		return nil
	}).DoOnComplete(func() {
		// will be called on successful completion of the stream
		fmt.Println("Word counter ended.")
	}).DoOnError(func(err error) {
		// will be called if a error occurs
		fmt.Println(err)
	}).DoFinally(func(s rx.SignalType) {
		// will always be called
		wg.Done()
	}).Subscribe(context.Background())

	// wait until the stream has finished
	wg.Wait()
}

// wordCount function
func wordCount(value string) int {
	words := strings.Fields(value)
	return len(words)
}
