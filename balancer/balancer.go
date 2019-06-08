package balancer

import (
	"io"

	"github.com/rsocket/rsocket-go"
)

type Balancer interface {
	io.Closer
	Put(client rsocket.Client)
	PutLabel(label string, client rsocket.Client)
	Next() rsocket.Client
	OnLeave(fn func(label string))
}
