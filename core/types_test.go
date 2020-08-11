package core_test

import (
	"testing"

	"github.com/rsocket/rsocket-go/core"
	"github.com/stretchr/testify/assert"
)

func TestFrameFlag_String(t *testing.T) {
	f := core.FlagNext | core.FlagComplete | core.FlagFollow | core.FlagMetadata | core.FlagIgnore
	assert.True(t, f.String() != "")
}
