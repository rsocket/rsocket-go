package common

import (
	"math/rand"
	"sync"
	"time"
)

var (
	defaultRand      = rand.New(rand.NewSource(time.Now().UnixNano()))
	defaultRandMutex = &sync.Mutex{}
)

// RandFloat64 returns a random float64.
func RandFloat64() float64 {
	defaultRandMutex.Lock()
	defer defaultRandMutex.Unlock()
	return defaultRand.Float64()
}

// RandIntn returns a random int.
func RandIntn(n int) int {
	defaultRandMutex.Lock()
	defer defaultRandMutex.Unlock()
	return defaultRand.Intn(n)
}
