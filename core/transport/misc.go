package transport

import (
	"context"
	"net/http"
	"strings"
)

func isClosedErr(err error) bool {
	if err == nil {
		return false
	}
	if err == http.ErrServerClosed {
		return true
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}

type ToClientTransport = func(context.Context) (*Transport, error)
type ToServerTransport = func(context.Context) (ServerTransport, error)
