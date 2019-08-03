package rx

import (
	"github.com/jjeffcaii/reactor-go"
)

func init() {
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
