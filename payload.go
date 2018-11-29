package rsocket

import (
	"fmt"
)

type Payload struct {
	Metadata []byte
	Data     []byte
}

func (p *Payload) String() string {
	return fmt.Sprintf("Payload{Metadata=%s, Data=%s}", string(p.Metadata), string(p.Data))
}

type SetupPayload struct {
	*Payload
}

func (p *SetupPayload) String() string {
	return fmt.Sprintf("SetupPayload{Metadata=%s, Data=%s}", string(p.Metadata), string(p.Data))
}
