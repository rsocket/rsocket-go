package rsocket

import (
	"fmt"
	"testing"
)

func TestByteBuffer(t *testing.T) {
	bf := borrowByteBuffer()
	_, _ = bf.Write([]byte("abcde"))
	fmt.Printf("%p\n", bf)
	fmt.Println("v:", string(bf.B))
	returnByteBuffer(bf)
	bf = borrowByteBuffer()
	fmt.Printf("%p\n", bf)
	fmt.Println("v:", string(bf.B))
}
