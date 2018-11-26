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
	err := p.decoder.Handle(func(frame Frame) error {
		t := frame.GetType()
		log.Printf("-----------------%s-----------------\n", frameTypeAlias[t])
		log.Println("streamID", frame.GetStreamID())
		log.Println("I:", frame.IsIgnore())
		log.Println("M:", frame.IsMetadata())
		switch t {
		case SETUP:
			f := &FrameSetup{
				Frame: frame,
			}
			log.Println("R:", f.IsResumeEnable())
			log.Println("L:", f.IsLease())
			log.Println("major:", f.GetMajor())
			log.Println("minor", f.GetMinor())
			log.Println("timeBetweenKeepalive:", f.GetTimeBetweenKeepalive())
			log.Println("maxLifetime:", f.GetMaxLifetime())
			log.Println("metadataMIME:", string(f.GetMetadataMIME()))
			log.Println("dataMIME:", string(f.GetDataMIME()))
			log.Println("payload:", string(f.GetPayload()))
			log.Println("---------------------------------------")
		case REQUEST_RESPONSE:
			f := &FrameRequestResponse{
				Frame: frame,
			}
			log.Println("F:", f.IsFollow())
			log.Println("metadataPayload:", string(f.GetMetadataPayload()))
			log.Println("payload:", string(f.GetPayload()))
		default:
			log.Println("frame:", frame)
		}
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
