package protocol

type ErrorCode uint32

func (p ErrorCode) String() string {
	return errorCodeAlias[p]
}

const (
	ERR_INVALID_SETUP     ErrorCode = 0x00000001
	ERR_UNSUPPORTED_SETUP ErrorCode = 0x00000002
	ERR_REJECTED_SETUP    ErrorCode = 0x00000003
	ERR_REJECTED_RESUME   ErrorCode = 0x00000004
	ERR_CONNECTION_ERROR  ErrorCode = 0x00000101
	ERR_CONNECTION_CLOSE  ErrorCode = 0x00000102
	ERR_APPLICATION_ERROR ErrorCode = 0x00000201
	ERR_REJECTED          ErrorCode = 0x00000202
	ERR_CANCELED          ErrorCode = 0x00000203
	ERR_INVALID           ErrorCode = 0x00000204
)

var (
	errorCodeAlias = map[ErrorCode]string{
		ERR_INVALID_SETUP:     "INVALID_SETUP",
		ERR_UNSUPPORTED_SETUP: "UNSUPPORTED_SETUP",
		ERR_REJECTED_SETUP:    "REJECTED_SETUP",
		ERR_REJECTED_RESUME:   "REJECTED_RESUME",
		ERR_CONNECTION_ERROR:  "CONNECTION_ERROR",
		ERR_CONNECTION_CLOSE:  "CONNECTION_CLOSE",
		ERR_APPLICATION_ERROR: "APPLICATION_ERROR",
		ERR_REJECTED:          "REJECTED",
		ERR_CANCELED:          "CANCELED",
		ERR_INVALID:           "INVALID",
	}
)

type FrameError struct {
	Frame
}

func (p FrameError) ErrorCode() ErrorCode {
	return ErrorCode(byteOrder.Uint32(p.Frame[frameHeaderLength : frameHeaderLength+4]))
}

func (p FrameError) ErrorData() []byte {
	return p.Frame[frameHeaderLength+4:]
}
