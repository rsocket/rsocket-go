package rsocket

import "fmt"

type frameCancel struct {
	*baseFrame
}

func (p *frameCancel) String() string {
	return fmt.Sprintf("frameCancel{%s}", p.header)
}

func createCancel(sid uint32) *frameCancel {
	return &frameCancel{
		&baseFrame{
			header: createHeader(sid, tCancel),
			body:   borrowByteBuffer(),
		},
	}

}
