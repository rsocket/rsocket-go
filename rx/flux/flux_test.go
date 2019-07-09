package flux_test

import (
	"context"
	"fmt"
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
	last.Release()
}

func TestFluxProcessorWithRequest(t *testing.T) {
	f := flux.Create(func(i context.Context, sink flux.Sink) {
		for i := 0; i < 3; i++ {
			sink.Next(payload.NewString("world", fmt.Sprintf("%d", i)))
		}
		sink.Complete()
	})

	s := rx.NewSubscriber(rx.OnNext(func(input payload.Payload) {
		log.Println("next:", input)
	}), rx.OnSubscribe(func(s rx.Subscription) {
		log.Println("onSub:")
		s.Request(1)
	}))

	f.DoFinally(func(s rx.SignalType) {
		println("finally")
	}).SubscribeOn(scheduler.Elastic()).SubscribeWith(context.Background(), s)

	time.Sleep(10 * time.Second)

}
