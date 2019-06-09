package transport

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/logger"
)

type wsConnection struct {
	c       *websocket.Conn
	counter *Counter
}

func (p *wsConnection) SetCounter(c *Counter) {
	p.counter = c
}

func (p *wsConnection) SetDeadline(deadline time.Time) error {
	return p.c.SetReadDeadline(deadline)
}

func (p *wsConnection) Read() (f framing.Frame, err error) {
	t, raw, err := p.c.ReadMessage()
	if err != nil {
		return
	}
	if t != websocket.BinaryMessage {
		logger.Warnf("omit non-binary messsage %d\n", t)
		return p.Read()
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
		common.ReturnByteBuffer(bf)
		return
	}
	base := framing.NewBaseFrame(header, bf)
	f, err = framing.NewFromBase(base)
	if err != nil {
		common.ReturnByteBuffer(bf)
		return
	}
	err = f.Validate()
	if err != nil {
		return
	}
	if logger.IsDebugEnabled() {
		logger.Debugf("<--- rcv: %s\n", f)
	}
	return
}

func (p *wsConnection) Write(frame framing.Frame) (err error) {
	err = p.c.WriteMessage(websocket.BinaryMessage, frame.Bytes())
	if err != nil {
		return
	}
	if logger.IsDebugEnabled() {
		logger.Debugf("---> snd: %s\n", frame)
	}
	return
}

func (p *wsConnection) Close() error {
	return p.c.Close()
}

func newWebsocketConnection(rawConn *websocket.Conn) *wsConnection {
	return &wsConnection{
		c: rawConn,
	}
}
