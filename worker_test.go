package rsocket

import (
	"fmt"
	"log"
	"sync/atomic"
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

func TestFoobar(t *testing.T) {
	var id uint32
	for range [10]struct{}{} {
		log.Println(2*(atomic.AddUint32(&id, 1)-1) + 1)
	}
}
