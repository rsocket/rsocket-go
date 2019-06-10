package rsocket

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlank(t *testing.T) {
	done := make(chan struct{})
	_, err := Connect().
		OnClose(func() {
			close(done)
		}).
		Transport("tcp://127.0.0.1:17878").Start(context.Background())
	require.NoError(t, err)
	<-done
}
