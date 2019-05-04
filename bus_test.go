package rsocket

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"

	. "github.com/stretchr/testify/require"
)

const metadataRegister = "register"

func TestBroker(t *testing.T) {
	bus := NewBus()
	err := Receive().
		Acceptor(func(setup payload.SetupPayload, sendingSocket CloseableRSocket) RSocket {
			// register socket to bus.
			metadata, _ := setup.MetadataUTF8()
			if metadata == metadataRegister {
				merge := struct {
					id string
					sk RSocket
				}{setup.DataUTF8(), sendingSocket}
				sendingSocket.OnClose(func() {
					bus.Remove(merge.id, merge.sk)
				})
				bus.Put(merge.id, merge.sk)
			}
			// bind responder: redirect to target socket.
			return NewAbstractSocket(
				RequestResponse(func(msg payload.Payload) rx.Mono {
					id, _ := msg.MetadataUTF8()
					sk, ok := bus.Get(id)
					if !ok {
						return rx.NewMono(func(ctx context.Context, sink rx.MonoProducer) {
							sink.Error(fmt.Errorf("MISSING_SERVICE_SOCKET_%s", id))
						})
					}
					return sk.RequestResponse(msg)
				}),
				RequestStream(func(msg payload.Payload) rx.Flux {
					id, _ := msg.MetadataUTF8()
					sk, ok := bus.Get(id)
					if !ok {
						return rx.NewFlux(func(ctx context.Context, producer rx.Producer) {
							producer.Error(fmt.Errorf("MISSING_SERVICE_SOCKET_%s", id))
						})
					}
					return sk.RequestStream(msg)
				}),
			)
		}).
		Transport("127.0.0.1:8888").
		Serve()
	panic(err)

}

func TestService(t *testing.T) {
	// Service A
	clientA1, err := Connect().
		SetupPayload(payload.NewString("A", metadataRegister)).
		Acceptor(func(socket RSocket) RSocket {
			return NewAbstractSocket(RequestResponse(func(msg payload.Payload) rx.Mono {
				return rx.JustMono(payload.NewString(msg.DataUTF8(), "RESPOND_FROM_A1"))
			}))
		}).
		Transport("tcp://127.0.0.1:8888").
		Start(context.Background())
	defer func() {
		_ = clientA1.Close()
	}()
	NoError(t, err)

	// Service A2
	clientA2, err := Connect().
		SetupPayload(payload.NewString("A", metadataRegister)).
		Acceptor(func(socket RSocket) RSocket {
			return NewAbstractSocket(RequestResponse(func(msg payload.Payload) rx.Mono {
				return rx.JustMono(payload.NewString(msg.DataUTF8(), "RESPOND_FROM_A2"))
			}))
		}).
		Transport("tcp://127.0.0.1:8888").
		Start(context.Background())
	defer func() {
		_ = clientA2.Close()
	}()
	NoError(t, err)

	// Service B
	clientB, err := Connect().
		SetupPayload(payload.NewString("B", metadataRegister)).
		Acceptor(func(socket RSocket) RSocket {
			return NewAbstractSocket(RequestResponse(func(msg payload.Payload) rx.Mono {
				return rx.JustMono(payload.NewString(msg.DataUTF8(), "RESPOND_FROM_B"))
			}), RequestStream(func(msg payload.Payload) rx.Flux {
				return rx.
					Range(0, 5).
					Map(func(n int) payload.Payload {
						return payload.NewString(fmt.Sprintf("STREAM_%02d: %s", n, msg.DataUTF8()), "RESPOND_FROM_B")
					})
			}))
		}).
		Transport("tcp://127.0.0.1:8888").
		Start(context.Background())
	defer func() {
		_ = clientB.Close()
	}()
	NoError(t, err)

	// 1. test RequestResponse
	log.Println("-----RequestResponse-----")
	clientA1.RequestResponse(payload.NewString("A1_TO_B", "B")).
		DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			m, _ := elem.MetadataUTF8()
			log.Printf("A1 -> B: data=%s, metadata=%s\n", elem.DataUTF8(), m)
		}).
		Subscribe(context.Background())
	clientA2.RequestResponse(payload.NewString("A2_TO_B", "B")).
		DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			m, _ := elem.MetadataUTF8()
			log.Printf("A2 -> B: data=%s, metadata=%s\n", elem.DataUTF8(), m)
		}).
		Subscribe(context.Background())
	for range [3]struct{}{} {
		clientB.RequestResponse(payload.NewString("B_TO_A", "A")).
			DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
				m, _ := elem.MetadataUTF8()
				log.Printf("B -> A: data=%s, metadata=%s\n", elem.DataUTF8(), m)
			}).
			Subscribe(context.Background())
	}
	// 2. test RequestStream
	log.Println("-----RequestStream-----")
	clientA1.RequestStream(payload.NewString("A1_TO_B_STREAM", "B")).
		DoOnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			m, _ := elem.MetadataUTF8()
			log.Printf("A1 -> B: data=%s, metadata=%s\n", elem.DataUTF8(), m)
		}).
		Subscribe(context.Background())

	// 3. test RequestResponse with cancel
	log.Println("-----RequestResponse_CANCEL-----")
	clientA2.RequestResponse(payload.NewString("A2_TO_B_TEST_CANCEL", "B")).
		DoOnCancel(func(ctx context.Context) {
			log.Println("cancel success!")
		}).
		DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			Fail(t, "it should be canceled.")
		}).
		Subscribe(context.Background(), rx.OnSubscribe(func(ctx context.Context, s rx.Subscription) {
			s.Cancel()
		}))

	// check leak
	time.Sleep(1 * time.Second)
	Equal(t, 0, common.CountByteBuffer(), "byte buffer leak")
}
