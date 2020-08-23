package lease_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/lease"
	"github.com/stretchr/testify/assert"
)

func TestSimpleLease_Next(t *testing.T) {
	l, err := lease.NewSimpleFactory(300*time.Millisecond, 100*time.Millisecond, 100*time.Millisecond, 1)
	assert.NoError(t, err, "create simple lease failed")
	lease, ok := l.Next(context.Background())
	assert.True(t, ok, "get next lease chan failed")
	next, ok := <-lease
	assert.True(t, ok, "get lease failed")
	fmt.Println(next)
}
