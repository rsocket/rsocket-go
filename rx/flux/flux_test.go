package flux_test

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/stretchr/testify/assert"
)

func TestJust(t *testing.T) {
	done := make(chan struct{})
	f := flux.Create(func(i context.Context, sink flux.Sink) {
		for i := 0; i < 10; i++ {
			sink.Next(payload.NewString(fmt.Sprintf("foo_%04d", i), fmt.Sprintf("bar_%04d", i)))
		}
		sink.Complete()
	})

	var su rx.Subscription

	f.
		DoOnNext(func(input payload.Payload) {
			log.Println("next:", input)
			su.Request(1)
		}).
		DoOnRequest(func(n int) {
			log.Println("request:", n)
		}).
		DoFinally(func(s rx.SignalType) {
			log.Println("finally")
			close(done)
		}).
		DoOnComplete(func() {
			log.Println("complete")
		}).
		Subscribe(context.Background(), rx.OnSubscribe(func(s rx.Subscription) {
			su = s
			su.Request(1)
		}))
	<-done
}

func TestProcessor(t *testing.T) {
	proc := flux.CreateProcessor()
	time.AfterFunc(1*time.Second, func() {
		proc.Next(payload.NewString("111", ""))
	})
	time.AfterFunc(2*time.Second, func() {
		proc.Next(payload.NewString("222", ""))
		proc.Complete()
	})

	done := make(chan struct{})

	proc.
		DoOnNext(func(input payload.Payload) {
			log.Println("next:", input)
		}).
		DoFinally(func(s rx.SignalType) {
			close(done)
		}).
		Subscribe(context.Background())
	<-done
}

func TestSwitchOnFirst(t *testing.T) {
	flux.Create(func(ctx context.Context, s flux.Sink) {
		s.Next(payload.NewString("5", ""))
		for i := 0; i < 10; i++ {
			s.Next(payload.NewString(fmt.Sprintf("%d", i), ""))
		}
		s.Complete()
	}).SwitchOnFirst(func(s flux.Signal, f flux.Flux) flux.Flux {
		v, ok := s.Value()
		if !ok {
			return f
		}
		first, _ := strconv.Atoi(v.DataUTF8())
		return f.Filter(func(input payload.Payload) bool {
			n, _ := strconv.Atoi(input.DataUTF8())
			return n > first
		})
	}).Subscribe(context.Background(), rx.OnNext(func(input payload.Payload) {
		fmt.Println("next:", input.DataUTF8())
	}))
}

func TestFluxRequest(t *testing.T) {
	f := flux.Create(func(ctx context.Context, s flux.Sink) {
		for i := 0; i < 10; i++ {
			s.Next(payload.NewString(fmt.Sprintf("DD_%04d", i), ""))
		}
		s.Complete()
	})

	var su rx.Subscription

	sub := rx.NewSubscriber(
		rx.OnNext(func(input payload.Payload) {
			log.Println("onNext:", input)
			su.Request(1)
		}),
		rx.OnComplete(func() {
			log.Println("complete")
		}),
		rx.OnSubscribe(func(s rx.Subscription) {
			su = s
			su.Request(1)
			log.Println("request:", 1)
		}),
	)

	f.SubscribeWith(context.Background(), sub)

}

func TestProxy_BlockLast(t *testing.T) {
	last, err := flux.Create(func(ctx context.Context, s flux.Sink) {
		for i := 0; i < 10; i++ {
			s.Next(payload.NewString(fmt.Sprintf("DD_%04d", i), ""))
		}
		s.Complete()
	}).BlockLast(context.Background())
	assert.NoError(t, err, "err occurred")
	log.Println(last)
}

func TestFluxProcessorWithRequest(t *testing.T) {
	f := flux.Create(func(i context.Context, sink flux.Sink) {
		for i := 0; i < 3; i++ {
			sink.Next(payload.NewString("world", fmt.Sprintf("%d", i)))
		}
		sink.Complete()
	})

	var su rx.Subscription

	sub := rx.NewSubscriber(
		rx.OnNext(func(input payload.Payload) {
			su.Request(1)
		}),
		rx.OnSubscribe(func(s rx.Subscription) {
			su = s
			su.Request(1)
		}),
	)

	done := make(chan struct{})

	f.
		DoFinally(func(s rx.SignalType) {
			close(done)
		}).
		SubscribeOn(scheduler.Elastic()).
		SubscribeWith(context.Background(), sub)
	<-done
}

func TestCreateFromChannel(t *testing.T) {
	payloads := make(chan payload.Payload)
	err := make(chan error)

	go func() {
		defer close(payloads)
		defer close(err)

		for i := 1; i <= 10000; i++ {
			p := payload.NewString(strconv.Itoa(i), strconv.Itoa(i))
			payloads <- p
		}
	}()

	background := context.Background()
	last, e := flux.
		CreateFromChannel(payloads, err).
		BlockLast(background)

	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, "10000", last.DataUTF8())

	m, _ := last.MetadataUTF8()
	assert.Equal(t, "10000", m)
}

func TestCreateFromChannelAndEmitError(t *testing.T) {
	payloads := make(chan payload.Payload)
	err := make(chan error)

	go func() {
		defer close(payloads)
		defer close(err)
		err <- errors.New("boom")
	}()

	background := context.Background()
	_, e := flux.
		CreateFromChannel(payloads, err).
		BlockLast(background)

	if e == nil {
		t.Fail()
	}
}

func TestCreateFromChannelWithNoEmitsOrErrors(t *testing.T) {
	payloads := make(chan payload.Payload)
	err := make(chan error)

	go func() {
		defer close(payloads)
		defer close(err)
	}()

	background := context.Background()
	_, e := flux.
		CreateFromChannel(payloads, err).
		BlockLast(background)

	if e != nil {
		t.Fail()
	}
}

func TestToChannel(t *testing.T) {
	payloads := make(chan payload.Payload)
	err := make(chan error)

	go func() {
		defer close(payloads)
		defer close(err)

		for i := 1; i <= 10; i++ {
			p := payload.NewString(strconv.Itoa(i), strconv.Itoa(i))
			payloads <- p
		}
	}()

	f := flux.CreateFromChannel(payloads, err)

	channel, chanerrors := flux.ToChannel(f, context.Background())

	var count int
loop:
	for {
		select {
		case _, o := <-channel:
			if o {
				count++
			} else {
				break loop
			}
		case err := <-chanerrors:
			if err != nil {
				t.Error(err)
				break loop
			}
		}
	}

	assert.Equal(t, 10, count)
}

func TestToChannelEmitError(t *testing.T) {
	payloads := make(chan payload.Payload)
	err := make(chan error)

	go func() {
		defer close(payloads)
		defer close(err)

		for i := 1; i <= 10; i++ {
			err <- errors.New("boom!")
		}
	}()

	f := flux.CreateFromChannel(payloads, err)

	channel, chanerrors := flux.ToChannel(f, context.Background())

loop:
	for {
		select {
		case _, o := <-channel:
			if o {
				t.Fail()
			} else {
				break loop
			}
		case err := <-chanerrors:
			if err != nil {
				break loop
			} else {
				t.Fail()
			}
		}
	}

}
