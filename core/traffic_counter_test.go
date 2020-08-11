package core_test

import (
	"sync"
	"testing"

	"github.com/rsocket/rsocket-go/core"
	"github.com/stretchr/testify/assert"
)

func TestTrafficCounter(t *testing.T) {
	const cycle = 1000
	const amount = 1000
	wg := sync.WaitGroup{}
	wg.Add(amount)
	c := core.NewTrafficCounter()
	for range [amount]struct{}{} {
		go func() {
			for range [cycle]struct{}{} {
				c.IncWriteBytes(1)
				c.IncReadBytes(1)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Equal(t, uint64(cycle*amount), c.WriteBytes())
	assert.Equal(t, uint64(cycle*amount), c.ReadBytes())
}
