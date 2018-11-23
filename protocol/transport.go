package protocol

type Transport interface {
	Send(frame *Frame) error
}

type ServerTransport interface {
	Transport
	Accept(acceptor func(conn *RConnection) error)
}
