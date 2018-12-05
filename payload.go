package rsocket

import "fmt"

type Payload interface {
	fmt.Stringer
	Metadata() []byte
	Data() []byte
}

type rawPayload struct {
	m []byte
	d []byte
}

func (p *rawPayload) String() string {
	return fmt.Sprintf("Payload{data=%s, metadata=%s}", string(p.d), string(p.m))
}

func (p *rawPayload) Metadata() []byte {
	return p.m
}

func (p *rawPayload) Data() []byte {
	return p.d
}

func CreatePayloadRaw(data []byte, metadata []byte) Payload {
	return &rawPayload{
		m: metadata,
		d: data,
	}
}

func CreatePayloadString(data string, metadata string) Payload {
	return CreatePayloadRaw([]byte(data), []byte(metadata))
}

type Version [2]uint16

func (p Version) Major() uint16 {
	return p[0]
}

func (p Version) Minor() uint16 {
	return p[1]
}

func (p Version) String() string {
	return fmt.Sprintf("%d.%d", p[0], p[1])
}

type SetupPayload interface {
	Payload
	Version() Version
}

type setupPayloadRaw struct {
	m []byte
	d []byte
	v Version
}

func (p *setupPayloadRaw) String() string {
	return fmt.Sprintf("SetupPayload{data=%s, metadata=%s, version=%s}", string(p.d), string(p.m), p.v)
}

func (p *setupPayloadRaw) Metadata() []byte {
	return p.m
}

func (p *setupPayloadRaw) Data() []byte {
	return p.d
}

func (p *setupPayloadRaw) Version() Version {
	return p.v
}

func newSetupPayload(v Version, data []byte, metadata []byte) SetupPayload {
	return &setupPayloadRaw{
		m: metadata,
		d: data,
		v: v,
	}
}
