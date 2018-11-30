package rsocket

type Payload interface {
	Metadata() []byte
	Data() []byte
}

type rawPayload struct {
	m []byte
	d []byte
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

type SetupPayload interface {
	Payload
	Version() Version
}

type setupPayloadRaw struct {
	m []byte
	d []byte
	v Version
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
