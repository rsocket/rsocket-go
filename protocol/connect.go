package protocol

type RConnection interface {
	Send(first *Frame, others ...*Frame) error
	Receive() (*Frame, error)
}
