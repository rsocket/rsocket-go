package balancer_test

import (
	"testing"

	"github.com/rsocket/rsocket-go/balancer"
	"github.com/stretchr/testify/assert"
)

var fakeGroupId = "fakeGroupId"

func TestGroup_Get(t *testing.T) {
	called := 0
	g := balancer.NewGroup(func() balancer.Balancer {
		called++
		return balancer.NewRoundRobinBalancer()
	})
	defer g.Close()
	for range [2]struct{}{} {
		b := g.Get(fakeGroupId)
		assert.NotNil(t, b)
		assert.Equal(t, 1, called)
	}
}
