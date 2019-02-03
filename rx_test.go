package rsocket

import (
	"fmt"
	"log"
	"testing"
)

func TestFlux_Basic(t *testing.T) {
	create := func(emitter Emitter) {
		for i := 0; i < 3; i++ {
			emitter.Next(NewPayloadString(fmt.Sprintf("hello_%d", i), fmt.Sprintf("world_%d", i)))
		}
		emitter.Complete()
	}
	done := make(chan struct{})
	NewFlux(create).
		DoOnNext(func(item Payload) {
			log.Println("onNext1:", item)
		}).
		DoOnNext(func(item Payload) {
			log.Println("onNext2:", item)
		}).
		DoFinally(func() {
			close(done)
		}).
		DoFinally(func() {
			log.Println("doFinally1")
		}).
		DoFinally(func() {
			log.Println("doFinally2")
		}).
		SubscribeOn(ElasticScheduler()).
		Subscribe(func(item Payload) {
			log.Println("subscribe", item)
		})
	<-done
}
