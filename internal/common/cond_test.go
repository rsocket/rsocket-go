package common_test

import (
	"context"
	"log"
	"runtime"
	"sync"
	"testing"

	"github.com/rsocket/rsocket-go/internal/common"
)

func TestNewCond(t *testing.T) {
	x := 0
	c := common.NewCond(&sync.Mutex{})
	done := make(chan bool)
	go func() {
		c.L.Lock()
		x = 1
		c.Wait(context.Background())
		if x != 2 {
			log.Fatal("want 2")
		}
		x = 3
		c.Broadcast()
		c.L.Unlock()
		done <- true
	}()
	go func() {
		c.L.Lock()
		for {
			if x == 1 {
				x = 2
				c.Broadcast()
				break
			}
			c.L.Unlock()
			runtime.Gosched()
			c.L.Lock()
		}
		c.L.Unlock()
		done <- true
	}()
	go func() {
		c.L.Lock()
		for {
			if x == 2 {
				c.Wait(context.Background())
				if x != 3 {
					log.Fatal("want 3")
				}
				break
			}
			if x == 3 {
				break
			}
			c.L.Unlock()
			runtime.Gosched()
			c.L.Lock()
		}
		c.L.Unlock()
		done <- true
	}()
	<-done
	<-done
	<-done
}
