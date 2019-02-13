package rsocket

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type RSocket struct {
	c           RConnection
	outStreamID uint32
	wp          workerPool

	hRQ  HandlerRQ
	hRS  HandlerRS
	hFNF HandlerFNF
	hMP  HandlerMetadataPush
	hRC  HandlerRC

	requestChannelsMap *sync.Map // sid -> flux
}

func (p *RSocket) HandleMetadataPush(h HandlerMetadataPush) {
	if h == nil {
		panic(ErrHandlerNil)
	}
	if p.hMP != nil {
		panic(ErrHandlerExist)
	}
	p.hMP = h
}

func (p *RSocket) HandleRequestResponse(h HandlerRQ) {
	if h == nil {
		panic(ErrHandlerNil)
	}
	if p.hRQ != nil {
		panic(ErrHandlerExist)
	}
	p.hRQ = h
}

func (p *RSocket) HandleRequestStream(h HandlerRS) {
	if h == nil {
		panic(ErrHandlerNil)
	}
	if p.hRS != nil {
		panic(ErrHandlerExist)
	}
	p.hRS = h
}

func (p *RSocket) HandleFireAndForget(h HandlerFNF) {
	if h == nil {
		panic(ErrHandlerNil)
	}
	if p.hFNF != nil {
		panic(ErrHandlerExist)
	}
	p.hFNF = h
}

func (p *RSocket) HandleRequestChannel(h HandlerRC) {
	if h == nil {
		panic(ErrHandlerNil)
	}
	if p.hRC != nil {
		panic(ErrHandlerExist)
	}
	p.hRC = h
}

func (p *RSocket) nextStreamID() uint32 {
	return atomic.AddUint32(&p.outStreamID, 2)
}

func newRSocket(c RConnection, wp workerPool) *RSocket {
	sk := &RSocket{
		c:                  c,
		outStreamID:        0,
		wp:                 wp,
		requestChannelsMap: &sync.Map{},
	}

	c.HandlePayload(func(f *framePayload) (err error) {
		// TODO: merge client and server codes.
		sid := f.header.StreamID()
		found, ok := sk.requestChannelsMap.Load(sid)
		if !ok {
			defer f.Release()
			return
		}
		// TODO: keep order, consider do in single scheduler
		flux := found.(*implFlux)
		fg := f.header.Flag()
		if fg.Check(FlagNext) {
			flux.Next(f)
		} else if fg.Check(FlagComplete) {
			defer f.Release()
			flux.Complete()
			sk.requestChannelsMap.Delete(sid)
		} else {
			panic(fmt.Errorf("illegal flag: %d", fg))
		}
		return
	})

	c.HandleRequestChannel(func(f *frameRequestChannel) (err error) {
		sid := f.header.StreamID()
		found, loaded := sk.requestChannelsMap.LoadOrStore(sid, newImplFlux())
		if loaded {
			panic(fmt.Errorf("duplicated request channel: streamID=%d", sid))
		}
		fluxIn := found.(*implFlux)

		// ensure RequestN
		sk.wp.Do(func() {
			_ = c.Send(createRequestN(sid, f.InitialRequestN()))
			fluxIn.Next(f)
		})
		// subscribe incoming channel payloads
		sk.wp.Do(func() {
			fluxOut := sk.hRC(fluxIn)
			if fluxOut != fluxIn {
				// auto release frame for each consumer
				fluxIn.setAfterConsumer(func(item Payload) {
					item.Release()
				})
			}
			fluxOut.
				DoFinally(func() {
					_ = c.Send(createPayloadFrame(sid, nil, nil, FlagComplete))
				}).
				Subscribe(toSender(sid, c, FlagNext))
		})
		return
	})

	c.HandleMetadataPush(func(f *frameMetadataPush) (err error) {
		sk.wp.Do(func() {
			defer f.Release()
			if sk.hMP == nil {
				return
			}
			sk.hMP(f.Metadata())
		})
		return
	})

	c.HandleFNF(func(frame *frameFNF) (err error) {
		sk.wp.Do(func() {
			defer frame.Release()
			if sk.hFNF == nil {
				return
			}
			sk.hFNF(frame)
		})
		return
	})

	c.HandleRequestStream(func(frame *frameRequestStream) (err error) {
		sk.wp.Do(func() {
			defer frame.Release()
			if sk.hRS == nil {
				return
			}
			sid := frame.header.StreamID()
			sk.hRS(frame).
				DoFinally(func() {
					_ = c.Send(createPayloadFrame(sid, nil, nil, FlagComplete))
				}).
				Subscribe(toSender(sid, c, FlagNext))
		})
		return
	})

	c.HandleRequestResponse(func(frame *frameRequestResponse) (err error) {
		sk.wp.Do(func() {
			if sk.hRQ == nil {
				defer frame.Release()
				return
			}
			res, err := sk.hRQ(frame)
			if err != nil {
				defer frame.Release()
				_ = c.Send(createError(frame.header.StreamID(), ErrorCodeApplicationError, []byte(err.Error())))
				return
			}
			if res == frame {
				// hack and reuse frame, reduce frame copy
				fg := FlagNext | FlagComplete
				if len(frame.Metadata()) > 0 {
					fg |= FlagMetadata
				}
				frame.setHeader(createHeader(frame.header.StreamID(), tPayload, fg))
				_ = c.Send(frame)
			} else {
				defer frame.Release()
				_ = c.Send(createPayloadFrame(frame.header.StreamID(), res.Data(), res.Metadata(), FlagNext|FlagComplete))
			}
			return
		})
		return
	})
	return sk
}

func toSender(sid uint32, c RConnection, fg Flags) Consumer {
	return func(item Payload) {
		switch v := item.(type) {
		case *framePayload:
			if v.header.Flag().Check(FlagMetadata) {
				fg |= FlagMetadata
			}
			v.setHeader(createHeader(sid, tPayload, fg))
			_ = c.Send(v)
		default:
			defer item.Release()
			_ = c.Send(createPayloadFrame(sid, item.Data(), item.Metadata(), fg))
		}
	}
}
