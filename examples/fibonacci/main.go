package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
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
	// create a handler that will be called when the server receives the RequestStream frame (FrameTypeRequestStream - 0x06)
	requestStreamHandler := rsocket.RequestStream(func(request payload.Payload) flux.Flux {

		// get number bytes from the request
		numberBytes := request.Data()
		// convert to uint64
		number := binary.LittleEndian.Uint64(numberBytes)

		fmt.Println("Number received: ", number)

		// check if number is part of a fibonacci sequence
		if !isFibonacci(number) {
			msg := "Number is NOT a part of a fibonacci sequence"
			return flux.Error(fmt.Errorf(msg))
		}

		fmt.Println("Number is part of a fibonacci sequence")

		// create a new flux, a publisher of data
		f := flux.Create(func(ctx context.Context, sink flux.Sink) {
			f := fibonacci()
			lastNumber := uint64(0)
			for lastNumber != number {
				lastNumber = f()
				// send next fibonacci number to client (peer)
				sink.Next(payload.NewString(fmt.Sprintf("%d", lastNumber), extension.TextPlain.String()))
			}

			// signal a completion of the stream of data
			sink.Complete()
		})

		return f
	})

	err := rsocket.Receive().
		OnStart(func() {
			// close the channel to singal that the server is ready
			close(readyCh)
		}).
		Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			// register a new request stream handler
			return rsocket.NewAbstractSocket(requestStreamHandler), nil
		}).
		// specify transport
		Transport(rsocket.TcpServer().SetAddr(":7878").Build()).
		// serve will block execution unless an error occurred
		Serve(context.Background())

	panic(err)
}

func client() {
	// Start a client connection
	tp := rsocket.TcpClient().SetHostAndPort("127.0.0.1", 7878).Build()
	client, err := rsocket.Connect().Transport(tp).Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// just something to help us put an integer into byte slice
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(number))

	// register request handler for RequestStream
	f := client.RequestStream(payload.New(b, []byte{}))

	// create a wait group so that the function does not return until the stream completes
	wg := sync.WaitGroup{}
	wg.Add(1)

	f.DoOnNext(func(input payload.Payload) error {
		// print each number in a stream
		fmt.Println(input.DataUTF8())
		return nil
	}).DoOnComplete(func() {
		// will be called on successful completion of the stream
		fmt.Println("Fibonacci sequence done")
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

// Fibonacci functions
func isFibonacci(number uint64) bool {
	return isSquare(5*number*number+4) || isSquare(5*number*number-4)
}

func isSquare(n uint64) bool {
	return n > 0 && math.Mod(math.Sqrt(float64(n)), 1) == 0
}

func fibonacci() func() uint64 {
	x, y := uint64(0), uint64(1)
	return func() uint64 {
		x, y = y, x+y
		return x
	}
}
