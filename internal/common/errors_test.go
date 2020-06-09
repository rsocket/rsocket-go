package common_test

import (
	"math"
	"testing"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestErrorCode_String(t *testing.T) {
	all := []common.ErrorCode{
		common.ErrorCodeInvalidSetup,
		common.ErrorCodeUnsupportedSetup,
		common.ErrorCodeRejectedSetup,
		common.ErrorCodeRejectedResume,
		common.ErrorCodeConnectionError,
		common.ErrorCodeConnectionClose,
		common.ErrorCodeApplicationError,
		common.ErrorCodeRejected,
		common.ErrorCodeCanceled,
		common.ErrorCodeInvalid,
	}
	for _, code := range all {
		assert.NotEqual(t, "UNKNOWN", code.String())
	}
	assert.Equal(t, "UNKNOWN", common.ErrorCode(math.MaxUint32).String())
}
