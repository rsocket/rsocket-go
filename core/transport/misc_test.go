package transport

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsClosedErr(t *testing.T) {
	assert.False(t, isClosedErr(nil))
	assert.False(t, isClosedErr(errors.New("fake error")))
	assert.True(t, isClosedErr(http.ErrServerClosed))
	assert.True(t, isClosedErr(errors.New("use of closed network connection")))
}
