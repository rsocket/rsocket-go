package tpfactory

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core/transport"
)

var (
	ErrConflictTransport    = errors.New("conflict transport")
	ErrUnavailableTransport = errors.New("transport is unavailable")
)

type TransportFactory struct {
	ch    chan *transport.Transport
	alive *transport.Transport
	mu    sync.RWMutex
}

func (tf *TransportFactory) Destroy() *transport.Transport {
	exist := tf.Reset()
	close(tf.ch)
	return exist
}

func NewTransportFactory() *TransportFactory {
	return &TransportFactory{
		ch: make(chan *transport.Transport, 1),
	}
}

func (tf *TransportFactory) Reset() (old *transport.Transport) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	if tf.alive != nil {
		old = tf.alive
		tf.alive = nil
	}
	select {
	case next, ok := <-tf.ch:
		if ok {
			old = next
		}
	default:
	}
	return
}

func (tf *TransportFactory) Get(ctx context.Context, block bool) (*transport.Transport, error) {
	tf.mu.RLock()
	alive := tf.alive
	tf.mu.RUnlock()

	if alive != nil {
		return alive, nil
	}

	if block {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case tp := <-tf.ch:
			tf.alive = tp
			return tp, nil
		}
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case tp := <-tf.ch:
		tf.alive = tp
		return tp, nil
	default:
		return nil, ErrUnavailableTransport
	}
}

func (tf *TransportFactory) Set(tp *transport.Transport) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	if tf.alive != nil {
		return ErrConflictTransport
	}
	select {
	case tf.ch <- tp:
		return nil
	default:
		return ErrConflictTransport
	}
}
