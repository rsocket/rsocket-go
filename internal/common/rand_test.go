package common_test

import (
	"regexp"
	"testing"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestRandAlphanumeric(t *testing.T) {
	s := common.RandAlphanumeric(10)
	r := regexp.MustCompile("^[a-zA-Z0-9]{10}$")
	assert.True(t, r.MatchString(s))
	s = common.RandAlphanumeric(0)
	assert.Empty(t, s)
}

func TestRandAlphabetic(t *testing.T) {
	s := common.RandAlphabetic(10)
	r := regexp.MustCompile("^[a-zA-Z]{10}$")
	assert.True(t, r.MatchString(s))
	s = common.RandAlphabetic(0)
	assert.Empty(t, s)
}

func TestRandFloat64(t *testing.T) {
	f := common.RandFloat64()
	assert.True(t, f < 1 && f > 0)
}

func TestRandIntn(t *testing.T) {
	n := common.RandIntn(10)
	assert.True(t, n >= 0 && n < 10)
}
