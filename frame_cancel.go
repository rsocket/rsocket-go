package rsocket

import "fmt"

type frameCancel struct {
	*baseFrame
}

func (p *frameCancel) String() string {
	return fmt.Sprintf("frameCancel{%s}", p.header)
}
