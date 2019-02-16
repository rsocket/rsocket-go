package rsocket

import (
	"log"
	"testing"
)

func TestXXADFSDF(t *testing.T) {
	ch := make(chan int, 1)
	//ch <- 1
	//log.Println("add")
	log.Println("consume:", <-ch)
}
