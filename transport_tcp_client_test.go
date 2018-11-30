package rsocket

import (
	"context"
	"log"
	"sync"
	"testing"
)

func TestTCPClientTransport_Connect(t *testing.T) {
	tp := newTCPClientTransport()
	c, err := tp.Connect("127.0.0.1:7878")
	if err != nil {
		t.Error(err)
	}

	c.HandlePayload(func(payload *FramePayload) error {
		log.Println("data:", string(payload.Data()))
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.loopRcv(ctx); err != nil {
			log.Println(err)
		}
	}()
	go func() {
		if err := c.loopSnd(ctx); err != nil {
			log.Println(err)
		}
	}()

	setup := mkSetup([]byte("foo"), []byte("bar"), "binary", "binary", FlagMetadata)
	if err := c.Send(setup); err != nil {
		t.Error(err)
	}
	if err := c.Send(mkRequestResponse(1, []byte("ping~"), []byte("ping"), FlagMetadata)); err != nil {
		t.Error(err)
	}

	wg.Wait()
	cancel()
}
