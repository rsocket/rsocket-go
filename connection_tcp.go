package rsocket

import (
	"bufio"
	"context"
	"io"
	"log"
	"sync"
	"time"
)

var keepaliveHeader = createHeader(0, tKeepalive)

type tcpRConnection struct {
	c            io.ReadWriteCloser
	decoder      frameDecoder
	snd          chan Frame
	rcv          chan *baseFrame
	onceClose    *sync.Once
	closeWaiting *sync.WaitGroup

	// handlers
	onSetup           func(f *frameSetup) (err error)
	onFNF             func(f *frameFNF) (err error)
	onPayload         func(f *framePayload) (err error)
	onRequestStream   func(f *frameRequestStream) (err error)
	onRequestResponse func(f *frameRequestResponse) (err error)
	onRequestChannel  func(f *frameRequestChannel) (err error)
	onCancel          func(f *frameCancel) (err error)
	onError           func(f *frameError) (err error)
	onMetadataPush    func(f *frameMetadataPush) (err error)

	keepaliveTicker *time.Ticker
}

func (p *tcpRConnection) HandleRequestChannel(h func(f *frameRequestChannel) (err error)) {
	if h == nil {
		panic(ErrHandlerNil)
	}
	if p.onRequestChannel != nil {
		panic(ErrHandlerExist)
	}
	p.onRequestChannel = h
}

func (p *tcpRConnection) HandleMetadataPush(h func(f *frameMetadataPush) (err error)) {
	if h == nil {
		panic(ErrHandlerNil)
	}
	if p.onMetadataPush != nil {
		panic(ErrHandlerExist)
	}
	p.onMetadataPush = h
}

func (p *tcpRConnection) HandleFNF(h func(f *frameFNF) (err error)) {
	if h == nil {
		panic(ErrHandlerNil)
	}
	if p.onFNF != nil {
		panic(ErrHandlerExist)
	}
	p.onFNF = h
}

func (p *tcpRConnection) HandlePayload(h func(f *framePayload) (err error)) {
	if h == nil {
		panic(ErrHandlerNil)
	}
	if p.onPayload != nil {
		panic(ErrHandlerExist)
	}
	p.onPayload = h
}

func (p *tcpRConnection) HandleRequestStream(h func(f *frameRequestStream) (err error)) {
	if h == nil {
		panic(ErrHandlerNil)
	}
	if p.onRequestStream != nil {
		panic(ErrHandlerExist)
	}
	p.onRequestStream = h
}

func (p *tcpRConnection) HandleRequestResponse(h func(f *frameRequestResponse) (err error)) {
	if h == nil {
		panic(ErrHandlerNil)
	}
	if p.onRequestResponse != nil {
		panic(ErrHandlerExist)
	}
	p.onRequestResponse = h
}

func (p *tcpRConnection) HandleSetup(h func(f *frameSetup) (err error)) {
	if h == nil {
		panic(ErrHandlerNil)
	}
	if p.onSetup != nil {
		panic(ErrHandlerExist)
	}
	p.onSetup = h
}

func (p *tcpRConnection) PostFlight(ctx context.Context) {
	go func() {
		defer func() {
			_ = p.Close()
		}()
		p.loopSnd()
	}()
	go func() {
		defer func() {
			_ = p.Close()
		}()
		p.loopRcv()
	}()
	go func() {
		defer func() {
			_ = p.Close()
		}()
		_ = p.decoder.handle(ctx, func(raw []byte) error {
			defer func() {
				_ = recover()
			}()
			h := parseHeaderBytes(raw)
			bf := borrowByteBuffer()
			if _, err := bf.Write(raw[headerLen:]); err != nil {
				return err
			}
			f := &baseFrame{h, bf}
			p.rcv <- f
			return nil
		})
	}()
}

func (p *tcpRConnection) Send(first Frame, others ...Frame) (err error) {
	defer func() {
		if e, ok := recover().(error); ok {
			err = e
		}
	}()
	p.snd <- first
	for _, other := range others {
		p.snd <- other
	}
	return
}

func (p *tcpRConnection) Close() (err error) {
	p.onceClose.Do(func() {
		if p.keepaliveTicker != nil {
			p.keepaliveTicker.Stop()
		}
		close(p.rcv)
		close(p.snd)
		// TODO: release unfinished frame
		p.closeWaiting.Wait()
		err = p.c.Close()
	})
	return
}

func (p *tcpRConnection) loopRcv() {
	defer p.closeWaiting.Done()
	p.closeWaiting.Add(1)
	var stop bool
	for {
		if stop {
			break
		}
		select {
		case in, ok := <-p.rcv:
			if !ok {
				stop = true
				break
			}
			if err := p.onRcv(in); err != nil {
				log.Println("rcv frame failed:", err)
			}
		}
	}
}

func (p *tcpRConnection) loopSnd() {
	defer p.closeWaiting.Done()
	p.closeWaiting.Add(1)
	w := bufio.NewWriterSize(p.c, defaultBuffSize)
	var stop bool
	if p.keepaliveTicker != nil {
		for {
			if stop {
				break
			}
			select {
			case out, ok := <-p.snd:
				if !ok {
					stop = true
					break
				}
				p.onSnd(out, w)
			case _, ok := <-p.keepaliveTicker.C:
				if ok {
					p.onSnd(createKeepalive(0, nil, true), w)
				}
			}
		}
	} else {
		for {
			if stop {
				break
			}
			select {
			case out, ok := <-p.snd:
				if !ok {
					stop = true
					break
				}
				p.onSnd(out, w)
			}
		}
	}
}

func (p *tcpRConnection) onRcv(f *baseFrame) (err error) {
	var hasHandler bool
	switch f.header.Type() {
	case tSetup:
		hasHandler = p.onSetup != nil
		if hasHandler {
			err = p.onSetup(&frameSetup{f})
		}
	case tKeepalive:
		hasHandler = true
		err = p.onKeepalive(&frameKeepalive{f})
	case tRequestResponse:
		hasHandler = p.onRequestResponse != nil
		if hasHandler {
			err = p.onRequestResponse(&frameRequestResponse{f})
		}
	case tRequestFNF:
		hasHandler = p.onFNF != nil
		if hasHandler {
			err = p.onFNF(&frameFNF{f})
		}
	case tRequestStream:
		hasHandler = p.onRequestStream != nil
		if hasHandler {
			err = p.onRequestStream(&frameRequestStream{f})
		}
	case tRequestChannel:
		hasHandler = p.onRequestChannel != nil
		if hasHandler {
			err = p.onRequestChannel(&frameRequestChannel{f})
		}
	case tCancel:
		hasHandler = p.onCancel != nil
		if hasHandler {
			err = p.onCancel(&frameCancel{f})
		}
	case tPayload:
		hasHandler = p.onPayload != nil
		if hasHandler {
			err = p.onPayload(&framePayload{f})
		}
	case tMetadataPush:
		hasHandler = p.onMetadataPush != nil
		if hasHandler {
			err = p.onMetadataPush(&frameMetadataPush{f})
		}
	case tError:
		hasHandler = true
		err = p.onError(&frameError{f})
	default:
		err = ErrInvalidFrame
	}
	if !hasHandler {
		defer f.Release()
		log.Printf("missing frame handler: %s\n", f.header)
	}
	return
}

func (p *tcpRConnection) onSnd(frame Frame, w *bufio.Writer) {
	defer frame.Release()
	if _, err := newUint24(frame.Len()).WriteTo(w); err != nil {
		log.Println("send frame failed:", err)
		return
	}
	if _, err := frame.WriteTo(w); err != nil {
		log.Println("send frame failed:", err)
		return
	}
	err := w.Flush()
	if err != nil {
		log.Println("send frame failed:", err)
	}
}

func (p *tcpRConnection) onKeepalive(f *frameKeepalive) (err error) {
	if !f.header.Flag().Check(FlagRespond) {
		f.Release()
		return
	}
	f.header = keepaliveHeader
	return p.Send(f)
}

func newTcpRConnection(c io.ReadWriteCloser, keepaliveInterval time.Duration) *tcpRConnection {
	var ticker *time.Ticker
	if keepaliveInterval > 0 {
		ticker = time.NewTicker(keepaliveInterval)
	}
	return &tcpRConnection{
		c:               c,
		decoder:         newLengthBasedFrameDecoder(c),
		snd:             make(chan Frame, 128),
		rcv:             make(chan *baseFrame, 128),
		keepaliveTicker: ticker,
		closeWaiting:    &sync.WaitGroup{},
		onceClose:       &sync.Once{},
		onError: func(f *frameError) (err error) {
			defer f.Release()
			log.Printf("ERROR: code=%s,data=%s\n", f.ErrorCode(), string(f.ErrorData()))
			return nil
		},
	}
}
