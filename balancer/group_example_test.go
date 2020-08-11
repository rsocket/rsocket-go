package balancer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/mono"
)

func ExampleNewGroup() {
	group := NewGroup(func() Balancer {
		return NewRoundRobinBalancer()
	})
	defer func() {
		_ = group.Close()
	}()
	// Create a broker with resume.
	err := rsocket.Receive().
		Resume(rsocket.WithServerResumeSessionDuration(10 * time.Second)).
		Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			// Register service using Setup Metadata as service ID.
			if serviceID, ok := setup.MetadataUTF8(); ok {
				group.Get(serviceID).Put(sendingSocket)
			}
			// Proxy requests by group.
			return rsocket.NewAbstractSocket(rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
				requestServiceID, ok := msg.MetadataUTF8()
				if !ok {
					panic(errors.New("missing service ID in metadata"))
				}
				fmt.Println("[broker] redirect request to service", requestServiceID)
				upstream, _ := group.Get(requestServiceID).Next(context.Background())
				fmt.Println("[broker] choose upstream:", upstream)
				return upstream.RequestResponse(msg)
			})), nil
		}).
		Transport(rsocket.TcpServer().SetAddr(":7878").Build()).
		Serve(context.Background())
	if err != nil {
		panic(err)
	}
}
