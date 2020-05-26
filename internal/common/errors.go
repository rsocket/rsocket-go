package common

import "errors"

// ErrorCode is code for RSocket error.
type ErrorCode uint32

// CustomError provides a method of accessing code and data.
type CustomError interface {
	error
	// ErrorCode returns error code.
	ErrorCode() ErrorCode
	// ErrorData returns error data bytes.
	ErrorData() []byte
}

func (p ErrorCode) String() string {
	switch p {
	case ErrorCodeInvalidSetup:
		return "INVALID_SETUP"
	case ErrorCodeUnsupportedSetup:
		return "UNSUPPORTED_SETUP"
	case ErrorCodeRejectedSetup:
		return "REJECTED_SETUP"
	case ErrorCodeRejectedResume:
		return "REJECTED_RESUME"
	case ErrorCodeConnectionError:
		return "CONNECTION_ERROR"
	case ErrorCodeConnectionClose:
		return "CONNECTION_CLOSE"
	case ErrorCodeApplicationError:
		return "APPLICATION_ERROR"
	case ErrorCodeRejected:
		return "REJECTED"
	case ErrorCodeCanceled:
		return "CANCELED"
	case ErrorCodeInvalid:
		return "INVALID"
	default:
		return "UNKNOWN"
	}
}

const (
	// ErrorCodeInvalidSetup means the setup frame is invalid for the server.
	ErrorCodeInvalidSetup ErrorCode = 0x00000001
	// ErrorCodeUnsupportedSetup means some (or all) of the parameters specified by the client are unsupported by the server.
	ErrorCodeUnsupportedSetup ErrorCode = 0x00000002
	// ErrorCodeRejectedSetup means server rejected the setup, it can specify the reason in the payload.
	ErrorCodeRejectedSetup ErrorCode = 0x00000003
	// ErrorCodeRejectedResume means server rejected the resume, it can specify the reason in the payload.
	ErrorCodeRejectedResume ErrorCode = 0x00000004
	// ErrorCodeConnectionError means the connection is being terminated.
	ErrorCodeConnectionError ErrorCode = 0x00000101
	// ErrorCodeConnectionClose means the connection is being terminated.
	ErrorCodeConnectionClose ErrorCode = 0x00000102
	// ErrorCodeApplicationError means application layer logic generating a Reactive Streams onError event.
	ErrorCodeApplicationError ErrorCode = 0x00000201
	// ErrorCodeRejected means Responder reject it.
	ErrorCodeRejected ErrorCode = 0x00000202
	// ErrorCodeCanceled means the Responder canceled the request but may have started processing it (similar to REJECTED but doesn't guarantee lack of side-effects).
	ErrorCodeCanceled ErrorCode = 0x00000203
	// ErrorCodeInvalid means the request is invalid.
	ErrorCodeInvalid ErrorCode = 0x00000204
)

// Error defines.
var (
	ErrFrameLengthExceed  = errors.New("rsocket: frame length is greater than 24bits")
	ErrInvalidTransport   = errors.New("rsocket: invalid Transport")
	ErrInvalidFrame       = errors.New("rsocket: invalid frame")
	ErrInvalidContext     = errors.New("rsocket: invalid context")
	ErrInvalidFrameLength = errors.New("rsocket: invalid frame length")
	ErrReleasedResource   = errors.New("rsocket: resource has been released")
	ErrInvalidEmitter     = errors.New("rsocket: invalid emitter")
	ErrHandlerNil         = errors.New("rsocket: handler cannot be nil")
	ErrHandlerExist       = errors.New("rsocket: handler exists already")
	ErrSendFull           = errors.New("rsocket: frame send channel is full")
)
