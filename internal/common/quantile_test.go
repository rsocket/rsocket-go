package common

import (
	"log"
	"math"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMedianQuantile_Insert(t *testing.T) {
	const totals = 100
	var sumErr float64
	var maxErr float64
	var minErr float64
	for range [totals]struct{}{} {
		n := 100 * 1024
		range2 := math.MaxInt32 >> 16
		median := NewMedianQuantile()
		data := make([]int, n)
		for i := 0; i < n; i++ {
			var x int
			v := range2/2 + int(float32(range2)/5*defaultRand.Float32())
			if v > 0 {
				x = v
			}
			data[i] = x
			median.Insert(float64(x))
		}
		sort.Sort(sort.IntSlice(data))
		expected := data[len(data)/2]

		estimation := median.Estimation()
		nowErr := math.Abs(float64(expected)-float64(estimation)) / float64(expected)

		sumErr += nowErr
		maxErr = math.Max(maxErr, nowErr)
		minErr = math.Min(minErr, nowErr)

		assert.True(t, nowErr < 0.02, "p50=%.4f, real=%d, error=%.4f\n", estimation, expected, nowErr)

	}
	log.Printf("error avg = %f in range [%f,%f]", sumErr/totals, minErr, maxErr)
}
