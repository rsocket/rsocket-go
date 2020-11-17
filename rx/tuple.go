package rx

import (
	"errors"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/tuple"
	"github.com/rsocket/rsocket-go/payload"
)

var errWrongTupleType = errors.New("tuple value must be a payload")

func IsWrongTupleTypeError(err error) bool {
	return err == errWrongTupleType
}

func NewTuple(t tuple.Tuple) Tuple {
	return Tuple{inner: t}
}

type Tuple struct {
	inner tuple.Tuple
}

func (t Tuple) First() (payload.Payload, error) {
	return t.innerReturn(t.inner.First())
}

func (t Tuple) Second() (payload.Payload, error) {
	return t.innerReturn(t.inner.Second())
}

func (t Tuple) Last() (payload.Payload, error) {
	return t.innerReturn(t.inner.Last())
}

func (t Tuple) Get(index int) (payload.Payload, error) {
	return t.innerReturn(t.inner.Get(index))
}

func (t Tuple) Len() int {
	return t.inner.Len()
}

func (t Tuple) ForEach(callback func(payload.Payload, error) bool) {
	t.inner.ForEach(func(v reactor.Any, e error) bool {
		if v == nil {
			return callback(nil, e)
		}
		p, ok := v.(payload.Payload)
		if ok {
			return callback(p, e)
		}
		return callback(nil, errWrongTupleType)
	})
}

func (t Tuple) ForEachWithIndex(callback func(payload.Payload, error, int) bool) {
	t.inner.ForEachWithIndex(func(v reactor.Any, e error, index int) bool {
		if v == nil {
			return callback(nil, e, index)
		}
		p, ok := v.(payload.Payload)
		if ok {
			return callback(p, e, index)
		}
		return callback(nil, errWrongTupleType, index)
	})
}

func (t Tuple) innerReturn(value reactor.Any, err error) (payload.Payload, error) {
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	p, ok := value.(payload.Payload)
	if ok {
		return p, nil
	}
	return nil, errWrongTupleType
}
