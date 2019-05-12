package transport

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/common/logger"
	"github.com/rsocket/rsocket-go/framing"
)

type wsConnection struct {
	c         *websocket.Conn
	snd       chan framing.Frame
	handler   func(ctx context.Context, frame framing.Frame) error
	onceClose *sync.Once
	done      chan struct{}

	kaEnable bool
	ka       *keepaliver
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
		_ = p.ka.Close()
		err = p.c.Close()
	})
	return
}

func (p *wsConnection) onRcv(ctx context.Context, f *framing.BaseFrame) (err error) {
	p.ka.Heartbeat()
	var frame framing.Frame
	frame, err = unwindFrame(f)
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

	if setupFrame, ok := frame.(*framing.FrameSetup); ok {
		interval := setupFrame.TimeBetweenKeepalive()
		p.ka.Reset(interval)
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

	var stop bool
	for {
		if stop {
			break
		}
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("send loop exit: %s\n", err)
			}
			return
		case t := <-p.ka.C():
			if p.ka.IsDead(t) {
				logger.Errorf("keepalive failed: remote connection is dead\n")
				stop = true
				continue
			}
			if !p.kaEnable {
				continue
			}
			if err := p.c.WriteMessage(websocket.BinaryMessage, getKeepaliveBytes(false)); err != nil {
				logger.Errorf("send frame failed: %s\n", err)
			}
		case out, ok := <-p.snd:
			if !ok {
				stop = true
				break
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
			err = errIncompleteHeader
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

func newWebsocketConnection(c *websocket.Conn, keepaliveInterval, keepaliveMaxLifetime time.Duration, reportKeepalive bool) *wsConnection {
	return &wsConnection{
		c:         c,
		onceClose: &sync.Once{},
		done:      make(chan struct{}),
		snd:       make(chan framing.Frame, common.DefaultTCPSendChanSize),
		kaEnable:  reportKeepalive,
		ka:        newKeepaliver(keepaliveInterval, keepaliveMaxLifetime),
	}
}
