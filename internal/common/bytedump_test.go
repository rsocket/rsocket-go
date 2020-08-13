package common

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestAppendPrettyHexDump(t *testing.T) {
	b := make([]byte, 100)
	rand.Read(b)
	s, _ := PrettyHexDump(b)
	fmt.Println(s)
}
