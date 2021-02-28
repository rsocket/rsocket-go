package loopywriter_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/loopywriter"
	"github.com/rsocket/rsocket-go/internal/tpfactory"
	"github.com/rsocket/rsocket-go/test"
	"github.com/stretchr/testify/assert"
)

func TestLoopyWriter_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	conn := test.NewMockConn(ctrl)
	conn.EXPECT().Write(gomock.Any()).Return(nil).MinTimes(1)
	conn.EXPECT().Flush().Return(nil).MinTimes(1)

	tp := transport.NewTransport(conn)

	tf := tpfactory.NewTransportFactory()
	_ = tf.Set(tp)

	done := make(chan struct{})
	q := loopywriter.NewCtrlQueue(done)
	c := core.NewTrafficCounter()
	w := loopywriter.NewLoopyWriter(q, false, c)

	go func() {
		defer close(done)
		for range [3]struct{}{} {
			q.Enqueue(nextFrame())
		}
	}()

	err := w.Run(context.Background(), 1*time.Second, tf)
	assert.NoError(t, err)
}

func nextFrame() core.WriteableFrame {
	return framing.NewWriteablePayloadFrame(1, []byte("foo"), []byte("bar"), 0)
}
