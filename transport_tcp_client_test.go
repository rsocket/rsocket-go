package rsocket

import (
	"context"
	"log"
	"sync"
	"testing"
)

func TestTCPClientTransport_Connect(t *testing.T) {
	tp := newTCPClientTransport("127.0.0.1", 8000)
	c, err := tp.Connect()
	if err != nil {
		t.Error(err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	c.HandlePayload(func(payload *FramePayload) error {
		defer wg.Done()
		log.Println("data:", string(payload.Data()))
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	c.PostFlight(ctx)

	setup := mkSetup([]byte("foo"), []byte("bar"), []byte("binary"), []byte("binary"), FlagMetadata)
	if err := c.Send(setup); err != nil {
		t.Error(err)
	}
	if err := c.Send(mkRequestResponse(1, []byte("ping~"), []byte("ping"), FlagMetadata)); err != nil {
		t.Error(err)
	}
	wg.Wait()
	cancel()
}
