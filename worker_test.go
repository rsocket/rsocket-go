package rsocket

import (
	"fmt"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	pool := newWorkerPool(5)

	pool.Do(func() {
		time.Sleep(10 * time.Millisecond)
		fmt.Println("wahaha1")
	})
	pool.Do(func() {
		time.Sleep(10 * time.Millisecond)
		fmt.Println("wahaha2")
	})
	time.Sleep(100 * time.Millisecond)
}
