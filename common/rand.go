package common

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

var (
	defaultRand      = rand.New(rand.NewSource(time.Now().UnixNano()))
	defaultRandMutex = &sync.Mutex{}
	alphanumeric     = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

// RandFloat64 returns a random float64.
func RandFloat64() (v float64) {
	defaultRandMutex.Lock()
	v = defaultRand.Float64()
	defaultRandMutex.Unlock()
	return
}

// RandIntn returns a random int.
func RandIntn(n int) (v int) {
	defaultRandMutex.Lock()
	v = defaultRand.Intn(n)
	defaultRandMutex.Unlock()
	return
}

// RandAlphabetic returns a string with random alphabets.
func RandAlphabetic(n int) (s string) {
	if n < 1 {
		return
	}
	bu := strings.Builder{}
	for i, l := 0, len(alphanumeric)-10; i < n; i++ {
		bu.WriteByte(alphanumeric[RandIntn(l)])
	}
	s = bu.String()
	return
}

// RandAlphanumeric returns a string with random alphabets and numbers.
func RandAlphanumeric(n int) (s string) {
	if n < 1 {
		return
	}
	bu := strings.Builder{}
	for i, l := 0, len(alphanumeric); i < n; i++ {
		bu.WriteByte(alphanumeric[RandIntn(l)])
	}
	s = bu.String()
	return
}
