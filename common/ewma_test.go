package common

import (
	"log"
	"testing"
	"time"
)

func TestEwma(t *testing.T) {
	ewma := NewEwma(1, time.Minute, float64(1*time.Microsecond)/float64(time.Second))
	for range [100]struct{}{} {
		time.Sleep(10 * time.Millisecond)
		ewma.Insert(RandFloat64())
		log.Println("ewma:", ewma)
	}
}
