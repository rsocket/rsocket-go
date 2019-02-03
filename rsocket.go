package rsocket

import "errors"

var (
	ErrInvalidTransport   = errors.New("rsocket: invalid transport")
	ErrInvalidFrame       = errors.New("rsocket: invalid frame")
	ErrInvalidContext     = errors.New("rsocket: invalid context")
	ErrInvalidFrameLength = errors.New("rsocket: invalid frame length")
	ErrReleasedResource   = errors.New("rsocket: resource has been released")
	ErrInvalidEmitter     = errors.New("rsocket: invalid emitter")
	ErrHandlerNil         = errors.New("rsocket: handler cannot be nil")
	ErrHandlerExist       = errors.New("rsocket: handler exists already")
)
