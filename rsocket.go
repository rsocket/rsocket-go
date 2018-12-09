package rsocket

import "errors"

var (
	ErrInvalidTransport = errors.New("rsocket: invalid transport")
	ErrInvalidFrame     = errors.New("rsocket: invalid frame")
)
