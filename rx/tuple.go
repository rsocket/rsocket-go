package rx

import (
	"errors"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/tuple"
	"github.com/rsocket/rsocket-go/payload"
)

var errWrongTupleType = errors.New("tuple value must be a payload")

// IsWrongTupleTypeError returns true if target error is type of wrong tuple type.
func IsWrongTupleTypeError(err error) bool {
	return err == errWrongTupleType
}

// NewTuple returns a new Tuple.
func NewTuple(t ...*reactor.Item) Tuple {
	return t
}

// Tuple is a container contains multiple items.
type Tuple []*reactor.Item

// First returns the first value or error.
func (t Tuple) First() (payload.Payload, error) {
	return t.convert(t.inner().First())
}

// Second returns the second value or error.
func (t Tuple) Second() (payload.Payload, error) {
	return t.convert(t.inner().Second())
}

// Last returns the last value or error.
func (t Tuple) Last() (payload.Payload, error) {
	return t.convert(t.inner().Last())
}

// Get returns the value or error with custom index.
func (t Tuple) Get(index int) (payload.Payload, error) {
	return t.convert(t.inner().Get(index))
}

// GetValue returns the value with custom index.
func (t Tuple) GetValue(index int) payload.Payload {
	if v := t.inner().GetValue(index); v != nil {
		return v.(payload.Payload)
	}
	return nil
}

// Len returns the length of Tuple.
func (t Tuple) Len() int {
	return t.inner().Len()
}

// ForEach visits each item in the Tuple.
func (t Tuple) ForEach(callback func(payload.Payload, error) bool) {
	t.inner().ForEach(func(v reactor.Any, e error) bool {
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

// ForEachWithIndex visits each item in the Tuple with index.
func (t Tuple) ForEachWithIndex(callback func(payload.Payload, error, int) bool) {
	t.inner().ForEachWithIndex(func(v reactor.Any, e error, index int) bool {
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

// HasError returns true if this Tuple contains error.
func (t Tuple) HasError() bool {
	return t.inner().HasError()
}

// CollectValues collects values and returns a slice.
func (t Tuple) CollectValues() (values []payload.Payload) {
	for i := 0; i < len(t); i++ {
		next := t[i]
		if next == nil || next.E != nil || next.V == nil {
			continue
		}
		if v, ok := next.V.(payload.Payload); ok {
			values = append(values, v)
		}
	}
	return
}

func (t Tuple) convert(value reactor.Any, err error) (payload.Payload, error) {
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

func (t Tuple) inner() tuple.Tuple {
	return tuple.NewTuple(t...)
}
