package rsocket

import (
	"github.com/jjeffcaii/go-rsocket/protocol"
	"log"
	"sync/atomic"
)

type RSocket struct {
	c           protocol.RConnection
	handlerRQ   HandlerRQ
	outStreamID uint32
	handlersRQ  map[uint32]HandlerRQ
}

func (p *RSocket) RequestResponse(send *Payload, callback HandlerRQ) {
	sid := atomic.AddUint32(&p.outStreamID, 2)
	log.Printf("SND: streamID=%d\n", sid)
	p.handlersRQ[sid] = callback
	// TODO
}

func newRSocket(c protocol.RConnection, hRQ HandlerRQ) *RSocket {
	ret := &RSocket{
		c:           c,
		outStreamID: 0,
		handlerRQ:   hRQ,
		handlersRQ:  make(map[uint32]HandlerRQ),
	}
	c.HandleRequestResponse(func(frame *protocol.FrameRequestResponse) error {
		if ret.handlerRQ == nil {
			return nil
		}
		req := &Payload{
			Data:     frame.Payload(),
			Metadata: frame.Metadata(),
		}
		var out protocol.Frame
		res, err := ret.handlerRQ(req)
		if err != nil {
			out = protocol.NewFrameError(&protocol.BaseFrame{
				Type:     protocol.ERROR,
				StreamID: frame.StreamID(),
				Flags:    0,
			}, protocol.ERR_APPLICATION_ERROR, []byte(err.Error()))
		} else {
			out = protocol.NewPayload(&protocol.BaseFrame{
				StreamID: frame.StreamID(),
				Flags:    protocol.FlagMetadata | protocol.FlagComplete | protocol.FlagNext,
				Type:     protocol.PAYLOAD,
			}, res.Data, res.Metadata)
		}
		return c.Send(out)
	})
	return ret
}
