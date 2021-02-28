package tpfactory_test

import (
	"context"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/tpfactory"
	"github.com/stretchr/testify/assert"
)

func TestNewTransportFactory(t *testing.T) {
	tp := transport.NewTransport(nil)

	f := tpfactory.NewTransportFactory()
	assert.NoError(t, f.Set(tp), "should set transport successfully")

	setConflict := func() {
		assert.Error(t, f.Set(tp), "should set failed")
		assert.Equal(t, tpfactory.ErrConflictTransport, f.Set(tp), "should be conflict error")
	}

	setConflict()

	for range [3]struct{}{} {
		next, err := f.Get(context.Background(), true)
		assert.NoError(t, err, "should get transport successfully")
		assert.Equal(t, tp, next)
	}

	setConflict()

	f.Reset()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := f.Get(ctx, true)
	assert.Error(t, err)

	assert.NoError(t, f.Set(tp))
	f.Reset()

	_, err = f.Get(ctx, false)
	assert.Error(t, err)
}
