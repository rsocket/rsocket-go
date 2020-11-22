package mono_test

import (
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	. "github.com/rsocket/rsocket-go/rx/mono"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func TestProxy_Error(t *testing.T) {
	originErr := errors.New("error testing")
	errCount := atomic.NewInt32(0)
	_, err := Error(originErr).
		DoOnError(func(e error) {
			assert.Equal(t, originErr, e, "bad error")
			errCount.Inc()
		}).
		Block(context.Background())
	assert.Error(t, err, "should got error")
	assert.Equal(t, originErr, err, "bad blocked error")
	assert.Equal(t, int32(1), errCount.Load(), "error count should be 1")
}

func TestEmpty(t *testing.T) {
	res, err := Empty().Block(context.Background())
	assert.NoError(t, err, "an error occurred")
	assert.Nil(t, res, "result should be nil")
}

func TestJustOrEmpty(t *testing.T) {
	// Give normal payload
	res, err := JustOrEmpty(payload.NewString("hello", "world")).Block(context.Background())
	assert.NoError(t, err, "an error occurred")
	assert.NotNil(t, res, "result should not be nil")
	// Give nil payload
	res, err = JustOrEmpty(nil).Block(context.Background())
	assert.NoError(t, err, "an error occurred")
	assert.Nil(t, res, "result should be nil")
}

func TestJust(t *testing.T) {
	Just(payload.NewString("hello", "world")).
		Subscribe(context.Background(), rx.OnNext(func(i payload.Payload) error {
			log.Println("next:", i)
			return nil
		}))
}

func TestMono_Raw(t *testing.T) {
	Just(payload.NewString("hello", "world")).Raw()
}

func TestProxy_SubscribeOn(t *testing.T) {
	v, err := Create(func(i context.Context, sink Sink) {
		time.AfterFunc(time.Second, func() {
			sink.Success(payload.NewString("foo", "bar"))
		})
	}).
		SubscribeOn(scheduler.Parallel()).
		DoOnSuccess(func(i payload.Payload) error {
			log.Println("success:", i)
			return nil
		}).
		Block(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "foo", v.DataUTF8(), "bad data result")
	m, _ := v.MetadataUTF8()
	assert.Equal(t, "bar", m, "bad metadata result")
}

func TestProxy_Block(t *testing.T) {
	v, err := Just(payload.NewString("hello", "world")).Block(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "hello", v.DataUTF8())
	m, _ := v.MetadataUTF8()
	assert.Equal(t, "world", m)
}

func TestProcessor(t *testing.T) {
	p, s, d := NewProcessor(nil, nil)
	defer d.Dispose()
	time.AfterFunc(3*time.Second, func() {
		s.Success(payload.NewString("hello", "world"))
	})
	v, err := p.Block(context.Background())
	assert.NoError(t, err)
	log.Println("block:", v)
}

func TestProxy_Filter(t *testing.T) {
	Just(payload.NewString("hello", "world")).
		Filter(func(i payload.Payload) bool {
			return strings.EqualFold("hello_no", i.DataUTF8())
		}).
		DoOnSuccess(func(i payload.Payload) error {
			assert.Fail(t, "should never run here")
			return nil
		}).
		DoFinally(func(i rx.SignalType) {
			log.Println("finally:", i)
		}).
		Subscribe(context.Background())
}

func TestMap(t *testing.T) {
	fakeMono := Just(payload.NewString("hello", "world"))
	value, err := fakeMono.
		Map(func(p payload.Payload) (payload.Payload, error) {
			data := strings.ToUpper(p.DataUTF8())
			metadata, _ := p.MetadataUTF8()
			return payload.NewString(data, metadata), nil
		}).
		Map(func(p payload.Payload) (payload.Payload, error) {
			metadata, _ := p.MetadataUTF8()
			return payload.NewString(p.DataUTF8(), strings.ToUpper(metadata)), nil
		}).
		Block(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "HELLO", value.DataUTF8())
	metadata, _ := value.MetadataUTF8()
	assert.Equal(t, "WORLD", metadata)

	_, err = fakeMono.Map(func(p payload.Payload) (payload.Payload, error) {
		return nil, errors.New("fake error")
	}).Block(context.Background())
	assert.Error(t, err)
}

func TestFlatMap(t *testing.T) {
	res, err := Just(payload.NewString("foo", "")).
		FlatMap(func(p payload.Payload) Mono {
			return Create(func(ctx context.Context, sink Sink) {
				select {
				case <-ctx.Done():
					sink.Error(errors.New("cancelled"))
				case <-time.After(100 * time.Millisecond):
					sink.Success(payload.NewString("bar", ""))
				}
			}).SubscribeOn(scheduler.Parallel())
		}).
		Block(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "bar", res.DataUTF8())
}

func TestCreate(t *testing.T) {
	Create(func(i context.Context, sink Sink) {
		sink.Success(payload.NewString("hello", "world"))
	}).
		DoOnSuccess(func(i payload.Payload) error {
			log.Println("doOnNext:", i)
			return nil
		}).
		DoFinally(func(s rx.SignalType) {
			log.Println("doFinally:", s)
		}).
		Subscribe(context.Background(), rx.OnNext(func(i payload.Payload) error {
			log.Println("next:", i)
			return nil
		}))

	Create(func(i context.Context, sink Sink) {
		sink.Error(errors.New("foobar"))
	}).
		DoOnError(func(e error) {
			assert.Equal(t, "foobar", e.Error(), "bad error")
		}).
		DoOnSuccess(func(i payload.Payload) error {
			assert.Fail(t, "should never run here")
			return nil
		}).
		Subscribe(context.Background())
}

func TestTimeout(t *testing.T) {
	gen := func(ctx context.Context, sink Sink) {
		time.Sleep(100 * time.Millisecond)
		sink.Success(payload.NewString("foobar", ""))
	}
	_, err := Create(gen).Timeout(10 * time.Millisecond).Block(context.Background())
	assert.Error(t, err, "should return error")
	assert.True(t, reactor.IsCancelledError(err), "should be cancelled error")
}

func TestBlockRelease(t *testing.T) {
	input := (*mockPayload)(atomic.NewInt32(0))
	_, release, err := Just(input).BlockUnsafe(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int32(1), input.RefCnt())
	release()
	assert.Equal(t, int32(0), input.RefCnt())
}

func TestBlockClone(t *testing.T) {
	input := (*mockPayload)(atomic.NewInt32(0))
	v, err := Just(input).Block(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int32(0), input.RefCnt())
	assert.NotEqual(t, input, v)
}

func TestSwitchIfError(t *testing.T) {
	fakeErr := errors.New("fake error")

	validator := func(input Mono) {
		next, err := input.
			SwitchIfError(func(err error) Mono {
				assert.Equal(t, fakeErr, err)
				return Just(_fakePayload)
			}).
			Block(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, _fakePayload, next)
	}

	validator(Error(fakeErr))
	validator(ErrorOneshot(fakeErr))
}

func TestSwitchValueIfError(t *testing.T) {
	validator := func(input Mono) {
		next, err := input.SwitchValueIfError(_fakePayload).Block(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, _fakePayload, next)
	}
	fakeErr := errors.New("fake error")
	validator(Error(fakeErr))
	validator(ErrorOneshot(fakeErr))
}

type mockPayload atomic.Int32

func (m *mockPayload) IncRef() int32 {
	return (*atomic.Int32)(m).Inc()
}

func (m *mockPayload) RefCnt() int32 {
	return (*atomic.Int32)(m).Load()
}

func (m *mockPayload) Release() {
	(*atomic.Int32)(m).Dec()
}

func (m *mockPayload) Metadata() (metadata []byte, ok bool) {
	return
}

func (m *mockPayload) MetadataUTF8() (metadata string, ok bool) {
	return
}

func (m *mockPayload) Data() []byte {
	return []byte("foobar")
}

func (m *mockPayload) DataUTF8() string {
	return string(m.Data())
}
