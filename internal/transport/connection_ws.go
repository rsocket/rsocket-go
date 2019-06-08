package transport

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/logger"
)

type wsConnection struct {
	c         *websocket.Conn
	snd       chan framing.Frame
	handler   func(ctx context.Context, frame framing.Frame) error
	onceClose *sync.Once
	done      chan struct{}
	counter   *Counter
}

func (p *wsConnection) SetCounter(c *Counter) {
	p.counter = c
}

func (p *wsConnection) SetDeadline(deadline time.Time) error {
	panic("implement me")
}

func (p *wsConnection) Handle(handler func(ctx context.Context, frame framing.Frame) error) {
	p.handler = handler
}

func (p *wsConnection) Send(frame framing.Frame) (err error) {
	p.snd <- frame
	return
}

func (p *wsConnection) Write(frame framing.Frame) error {
	defer func() {
		frame.Release()
	}()
	if logger.IsDebugEnabled() {
		logger.Debugf("---> snd: %s\n", frame)
	}
	return p.c.WriteMessage(websocket.BinaryMessage, frame.Bytes())
}

func (p *wsConnection) Start(ctx context.Context) (err error) {
	defer func() {
		_ = p.Close()
	}()
	go func(ctx context.Context) {
		p.loopWrite(ctx)
	}(ctx)
	err = p.loopRead(ctx)
	return
}

func (p *wsConnection) Close() (err error) {
	p.onceClose.Do(func() {
		close(p.snd)
		<-p.done
		err = p.c.Close()
	})
	return
}

func (p *wsConnection) onRcv(ctx context.Context, f *framing.BaseFrame) (err error) {
	var frame framing.Frame
	frame, err = framing.NewFromBase(f)
	if err != nil {
		return
	}
	err = frame.Validate()
	if err != nil {
		return
	}

	if logger.IsDebugEnabled() {
		logger.Debugf("<--- rcv: %s\n", frame)
	}

	err = p.handler(ctx, frame)
	if err != nil {
		logger.Warnf("handle frame %s failed: %s\n", frame.Header().Type(), err.Error())
	}
	return
}

func (p *wsConnection) loopWrite(ctx context.Context) {
	defer func() {
		logger.Debugf("send loop exit\n")
		close(p.done)
	}()

L:
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("send loop exit: %s\n", err)
			}
			return
		case out, ok := <-p.snd:
			if !ok {
				break L
			}
			if err := p.Write(out); err != nil {
				logger.Errorf("send frame failed: %s\n", err)
			}
		}
	}
}

func (p *wsConnection) loopRead(ctx context.Context) (err error) {
	defer func() {
		_ = p.Close()
	}()

	var t int
	var raw []byte
	for {
		t, raw, err = p.c.ReadMessage()
		if err != nil {
			return err
		}
		if t != websocket.BinaryMessage {
			log.Println("unknown message type:", t)
			continue
		}
		// validate min length
		if len(raw) < framing.HeaderLen {
			err = ErrIncompleteHeader
			return
		}

		header := framing.ParseFrameHeader(raw)
		bf := common.BorrowByteBuffer()
		_, err = bf.Write(raw[framing.HeaderLen:])
		if err != nil {
			return
		}
		err = p.onRcv(ctx, framing.NewBaseFrame(header, bf))
		if err != nil {
			return
		}
	}
}

func newWebsocketConnection(rawConn *websocket.Conn) *wsConnection {
	return &wsConnection{
		c:         rawConn,
		onceClose: &sync.Once{},
		done:      make(chan struct{}),
		snd:       make(chan framing.Frame, 64),
	}
}
