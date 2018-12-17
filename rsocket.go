package rsocket

import "errors"

var (
	ErrInvalidTransport   = errors.New("rsocket: invalid transport")
	ErrInvalidFrame       = errors.New("rsocket: invalid frame")
	ErrInvalidContext     = errors.New("rsocket: invalid context")
	ErrInvalidFrameLength = errors.New("rsocket: invalid frame length")
)
