package rx_test

import (
	"errors"
	"testing"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/tuple"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/stretchr/testify/assert"
)

var (
	fakePayload = payload.NewString("fake", "payload")
	fakeError   = errors.New("fake error")
)

func TestTuple(t *testing.T) {
	rawTuple := tuple.NewTuple(&reactor.Item{V: fakePayload}, &reactor.Item{E: fakeError}, &reactor.Item{})
	tup := rx.NewTuple(rawTuple)
	v, err := tup.First()
	assert.NoError(t, err, "should not return error")
	assert.Equal(t, fakePayload, v, "should be fake payload")

	_, err = tup.Second()
	assert.Error(t, fakeError, err, "should return error")

	v, err = tup.Last()
	assert.NoError(t, err, "should not return error")
	assert.Nil(t, v, "payload should be nil")

	_, err = tup.Get(999)
	assert.Error(t, err, "should be not exist")

	var visits int
	tup.ForEach(func(p payload.Payload, err error) bool {
		visits++
		return false
	})
	assert.Equal(t, 1, visits)

	visits = 0
	tup.ForEach(func(p payload.Payload, err error) bool {
		visits++
		return true
	})
	assert.Equal(t, visits, tup.Len())

	visits = 0
	tup.ForEachWithIndex(func(p payload.Payload, err error, i int) bool {
		visits++
		return false
	})
	assert.Equal(t, 1, visits)

	visits = 0
	tup.ForEachWithIndex(func(p payload.Payload, err error, i int) bool {
		visits++
		return true
	})
	assert.Equal(t, visits, tup.Len())
}

func TestTupleWithWrongType(t *testing.T) {
	tup := rx.NewTuple(tuple.NewTuple(&reactor.Item{V: 1}))
	_, err := tup.First()
	assert.Error(t, err)
	assert.True(t, rx.IsWrongTupleTypeError(err))
	tup.ForEach(func(p payload.Payload, err error) bool {
		assert.Error(t, err)
		return true
	})
	tup.ForEachWithIndex(func(p payload.Payload, err error, i int) bool {
		assert.Error(t, err)
		return true
	})
}

func TestTuple_Empty(t *testing.T) {
	tup := tuple.NewTuple()
	assert.Zero(t, tup.Len())
}
