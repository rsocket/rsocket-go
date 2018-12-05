package rsocket

import (
	"context"
	"log"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cli, err := NewClient(
		"127.0.0.1",
		7878,
		WithSetupPayload([]byte("hello"), nil),
		WithKeepalive(2*time.Second, 3*time.Second, 3),
	)
	if err != nil {
		t.Error(err)
	}

	ctx := context.Background()
	go func() {
		cli.Start(ctx)
	}()
	cli.RequestResponse(CreatePayloadString("ping", "hahaha"), func(res Payload) {
		log.Printf("=====>%+v\n", res)
	})

	time.Sleep(1 * time.Minute)
}
