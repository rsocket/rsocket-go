package rsocket

import (
	"log"
	"sync/atomic"
)

type RSocket struct {
	c           RConnection
	handlerRQ   HandlerRQ
	handlerRS   HandlerRS
	handlerFNF  HandlerFNF
	outStreamID uint32
	handlersRQ  map[uint32]HandlerRQ
}

func (p *RSocket) HandleRequestResponse(h HandlerRQ) {
	p.handlerRQ = h
}

func (p *RSocket) HandleRequestStream(h HandlerRS) {
	p.handlerRS = h
}

func (p *RSocket) HandleFireAndForget(h HandlerFNF) {
	p.handlerFNF = h
}

func (p *RSocket) RequestResponse(send *Payload, callback HandlerRQ) {
	sid := atomic.AddUint32(&p.outStreamID, 2)
	log.Printf("SND: streamID=%d\n", sid)
	p.handlersRQ[sid] = callback
	// TODO
}

type emitterImpl struct {
	c        RConnection
	streamID uint32
}

func (p *emitterImpl) Error(err error) error {
	out := mkError(p.streamID, ERR_APPLICATION_ERROR, []byte(err.Error()))
	return p.c.Send(out)
}

func (p *emitterImpl) Next(payload Payload) error {
	fg := FlagNext
	if payload.Metadata() != nil {
		fg |= FlagMetadata
	}
	return p.c.Send(mkPayload(p.streamID, payload.Metadata(), payload.Data(), fg))
}

func (p *emitterImpl) Complete(payload Payload) error {
	fg := FlagNext | FlagComplete
	if payload.Metadata() != nil {
		fg |= FlagMetadata
	}
	return p.c.Send(mkPayload(p.streamID, payload.Metadata(), payload.Data(), fg))
}

func newRSocket(c RConnection) *RSocket {
	ret := &RSocket{
		c:           c,
		outStreamID: 0,
		handlersRQ:  make(map[uint32]HandlerRQ),
	}

	c.HandleFNF(func(frame *FrameFNF) (err error) {
		if ret.handlerFNF == nil {
			return nil
		}
		go func(sid uint32, payload Payload) {
			if err := ret.handlerFNF(payload); err != nil {
				log.Println("handle fnf failed:", err)
			}
		}(frame.StreamID(), CreatePayloadRaw(frame.Data(), frame.Metadata()))
		return nil
	})

	c.HandleRequestStream(func(frame *FrameRequestStream) (err error) {
		if ret.handlerRS == nil {
			return nil
		}
		go func(sid uint32, req Payload) {
			mp := &emitterImpl{
				c:        c,
				streamID: sid,
			}
			ret.handlerRS(req, mp)
		}(frame.StreamID(), CreatePayloadRaw(frame.Data(), frame.Metadata()))
		return nil
	})

	c.HandleRequestResponse(func(frame *FrameRequestResponse) error {
		if ret.handlerRQ == nil {
			return nil
		}
		go func(sid uint32, req Payload) {
			mh := &emitterImpl{
				streamID: sid,
				c:        c,
			}
			res, err := ret.handlerRQ(req)
			if err != nil {
				_ = mh.Error(err)
			} else {
				_ = mh.Complete(res)
			}
		}(frame.StreamID(), CreatePayloadRaw(frame.Data(), frame.Metadata()))
		return nil
	})
	return ret
}
