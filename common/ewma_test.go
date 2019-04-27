package common

import (
	"log"
	"testing"
	"time"
)

func TestEwma(t *testing.T) {
	var x float64 = 100
	ewma := NewEwma(1, time.Minute, float64(1*time.Microsecond)/float64(time.Second))
	for range [100]struct{}{} {
		ewma.Insert(x)
		x += 100
		log.Println("ewma:", ewma.Value())
		time.Sleep(3 * time.Second)
	}
}
