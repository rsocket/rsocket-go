package framing

import "github.com/rsocket/rsocket-go/core"

type writeableFrame struct {
	header      core.FrameHeader
	doneHandler func()
}

func newWriteableFrame(header core.FrameHeader) writeableFrame {
	return writeableFrame{
		header: header,
	}
}

func (t writeableFrame) Header() core.FrameHeader {
	return t.header
}

// Done can be invoked when a frame has been been processed.
func (t *writeableFrame) Done() {
	var h func()
	h, t.doneHandler = t.doneHandler, nil
	if h != nil {
		h()
	}
}

func (t *writeableFrame) HandleDone(h func()) {
	t.doneHandler = h
}
