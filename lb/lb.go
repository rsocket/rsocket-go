package lb

import (
	"io"

	"github.com/rsocket/rsocket-go"
)

type Balancer interface {
	io.Closer
	Push(client rsocket.Client)
	Next() (rsocket.Client, bool)
}
