package rsocket

type ErrorCode uint32

var errorCodeMap = map[ErrorCode]string{
	ErrorCodeInvalidSetup:     "INVALID_SETUP",
	ErrorCodeUnsupportedSetup: "UNSUPPORTED_SETUP",
	ErrorCodeRejectedSetup:    "REJECTED_SETUP",
	ErrorCodeRejectedResume:   "REJECTED_RESUME",
	ErrorCodeConnectionError:  "CONNECTION_ERROR",
	ErrorCodeConnectionClose:  "CONNECTION_CLOSE",
	ErrorCodeApplicationError: "APPLICATION_ERROR",
	ErrorCodeRejected:         "REJECTED",
	ErrorCodeCanceled:         "CANCELED",
	ErrorCodeInvalid:          "INVALID",
}

func (p ErrorCode) String() string {
	if s, ok := errorCodeMap[p]; ok {
		return s
	}
	return "UNKNOWN"
}

const (
	ErrorCodeInvalidSetup     ErrorCode = 0x00000001
	ErrorCodeUnsupportedSetup ErrorCode = 0x00000002
	ErrorCodeRejectedSetup    ErrorCode = 0x00000003
	ErrorCodeRejectedResume   ErrorCode = 0x00000004
	ErrorCodeConnectionError  ErrorCode = 0x00000101
	ErrorCodeConnectionClose  ErrorCode = 0x00000102
	ErrorCodeApplicationError ErrorCode = 0x00000201
	ErrorCodeRejected         ErrorCode = 0x00000202
	ErrorCodeCanceled         ErrorCode = 0x00000203
	ErrorCodeInvalid          ErrorCode = 0x00000204
)

type RError interface {
	error
	ErrorCode() ErrorCode
	ErrorData() []byte
}
