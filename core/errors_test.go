package core_test

import (
	"math"
	"testing"

	"github.com/rsocket/rsocket-go/core"
	"github.com/stretchr/testify/assert"
)

func TestErrorCode_String(t *testing.T) {
	all := []core.ErrorCode{
		core.ErrorCodeInvalidSetup,
		core.ErrorCodeUnsupportedSetup,
		core.ErrorCodeRejectedSetup,
		core.ErrorCodeRejectedResume,
		core.ErrorCodeConnectionError,
		core.ErrorCodeConnectionClose,
		core.ErrorCodeApplicationError,
		core.ErrorCodeRejected,
		core.ErrorCodeCanceled,
		core.ErrorCodeInvalid,
	}
	for _, code := range all {
		assert.NotEqual(t, "UNKNOWN", code.String())
	}
	assert.Equal(t, "UNKNOWN", core.ErrorCode(math.MaxUint32).String())
}
