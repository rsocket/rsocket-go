package protocol

import (
	"log"
	"net"
)

type TcpServerTransport struct {
	addr string
}

func (p *TcpServerTransport) Listen() {
	listener, err := net.Listen("tcp", p.addr)
	if err != nil {
		panic(err)
	}
	for {
		select {
		default:
			c, err := listener.Accept()
			if err != nil {
				log.Println("tcp listener break:", err)
				break
			}
			go func(rc *tcpRConnection) {
				if err := rc.poll(); err != nil {
					log.Println("tcp rconnection error:", err)
				}
			}(newTcpRConnection(c))
		}
	}
}

func NewTcpServerTransport(addr string) *TcpServerTransport {
	return &TcpServerTransport{
		addr: addr,
	}
}

type tcpRConnection struct {
	c       net.Conn
	decoder *tcpFrameDecoder
}

func (p *tcpRConnection) Close() error {
	return p.c.Close()
}

func (p *tcpRConnection) Send(first *Frame, others ...*Frame) error {
	panic("implement me")
}

func (p *tcpRConnection) Receive() (*Frame, error) {
	panic("implement me")
}

func (p *tcpRConnection) poll() error {
	defer func() {
		if err := p.Close(); err != nil {
			log.Println("close connection failed:", err)
		}
	}()
	err := p.decoder.Handle(func(frame *Frame) error {
		log.Printf("%s: %+v\n", frameTypeAlias[frame.Header.Type], frame)
		return nil
	})
	if err != nil {
		log.Println("poll stop:", err)
	}
	return err
}

func newTcpRConnection(c net.Conn) *tcpRConnection {
	return &tcpRConnection{
		c:       c,
		decoder: newTcpFrameDecoder(c),
	}
}
