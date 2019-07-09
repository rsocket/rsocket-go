package rx

import (
	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/hooks"
	"github.com/rsocket/rsocket-go/payload"
)

func init() {
	// Ensure that each Payload has been properly released.
	hooks.OnErrorDrop(func(e error) {
		if v, ok := e.(payload.Payload); ok {
			v.Release()
		}
	})
	hooks.OnNextDrop(func(v interface{}) {
		if vv, ok := v.(payload.Payload); ok {
			vv.Release()
		}
	})
}

const RequestMax = rs.RequestInfinite

type (
	// Disposable is a disposable resource.
	Disposable interface {
		// Dispose dispose current resource.
		Dispose()
		// IsDisposed returns true if it has been disposed.
		IsDisposed() bool
	}
)
