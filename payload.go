package rsocket

type Payload interface {
	Metadata() []byte
	Data() []byte
	Release()
}

func NewPayload(data []byte, metadata []byte) Payload {
	return createPayloadFrame(0, data, metadata)
}

func NewPayloadString(data string, metadata string) Payload {
	return NewPayload([]byte(data), []byte(metadata))
}
