package rsocket

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

func TestFlux_Basic(t *testing.T) {
	create := func(ctx context.Context, emitter Emitter) {
		for i := 0; i < 3; i++ {
			emitter.Next(NewPayloadString(fmt.Sprintf("hello_%d", i), fmt.Sprintf("world_%d", i)))
		}
		emitter.Complete()
	}
	done := make(chan struct{})
	NewFlux(create).
		DoOnNext(func(ctx context.Context, item Payload) {
			log.Println("onNext1:", item)
		}).
		DoOnNext(func(ctx context.Context, item Payload) {
			log.Println("onNext2:", item)
		}).
		DoFinally(func(ctx context.Context) {
			close(done)
		}).
		DoFinally(func(ctx context.Context) {
			log.Println("doFinally1")
		}).
		DoFinally(func(ctx context.Context) {
			log.Println("doFinally2")
		}).
		SubscribeOn(ElasticScheduler()).
		Subscribe(context.Background(), func(ctx context.Context, item Payload) {
			log.Println("subscribe", item)
		})
	<-done
}

func TestMono_All(t *testing.T) {
	ob := func(emitter MonoEmitter) {
		time.Sleep(1 * time.Millisecond)
		emitter.Success(NewPayloadString("hello", "world"))
	}

	done := make(chan struct{})
	NewMono(ob).
		DoFinally(func(ctx context.Context) {
			log.Println("finally 1")
			close(done)
		}).
		DoFinally(func(ctx context.Context) {
			log.Println("finally 2")
		}).
		DoOnSuccess(func(ctx context.Context, item Payload) {
			log.Println("success1:", item)
		}).
		DoOnNext(func(ctx context.Context, item Payload) {
			log.Println("next1:", item)
		}).
		DoOnSuccess(func(ctx context.Context, item Payload) {
			log.Println("success2:", item)
		}).
		DoOnNext(func(ctx context.Context, item Payload) {
			log.Println("next2:", item)
		}).
		SubscribeOn(ElasticScheduler()).
		Subscribe(context.Background(), func(ctx context.Context, item Payload) {
			log.Println("sub:", item)
		})
	<-done
}

func TestRx_Context(t *testing.T) {
	ctx := context.WithValue(context.Background(), "kkk", "vvv")
	fx := NewFlux(func(ctx context.Context, emitter Emitter) {
		for i := 0; i < 100; i++ {
			emitter.Next(NewPayloadString(fmt.Sprintf("foo_%d", i), "xxx"))
		}
		emitter.Complete()
	})
	done := make(chan struct{})
	fx.
		DoFinally(func(ctx context.Context) {
			close(done)
		}).
		SubscribeOn(ElasticScheduler()).Subscribe(ctx, func(ctx context.Context, item Payload) {
		vv := ctx.Value("kkk")
		log.Println("vv:", vv)
		log.Println("item:", item)
	})
	<-done
}
