package protocol

import (
	"log"
	"net"
)

type tcpRConnection struct {
	c       net.Conn
	decoder FrameDecoder
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
		t := frame.Type()
		log.Printf("-----------------%s-----------------\n", frameTypeAlias[t])
		log.Println("streamID", frame.StreamID())
		log.Println("I:", frame.IsIgnore())
		log.Println("M:", frame.IsMetadata())
		switch t {
		case SETUP:
			f := &FrameSetup{
				Frame: frame,
			}
			log.Println("R:", f.IsResumeEnable())
			log.Println("L:", f.IsLease())
			log.Println("major:", f.Major())
			log.Println("minor", f.Minor())
			log.Println("timeBetweenKeepalive:", f.TimeBetweenKeepalive())
			log.Println("maxLifetime:", f.MaxLifetime())
			log.Println("metadataMIME:", string(f.MetadataMIME()))
			log.Println("dataMIME:", string(f.DataMIME()))
			log.Println("metadata:", string(f.Metadata()))
			log.Println("payload:", string(f.Payload()))
		case REQUEST_RESPONSE:
			f := &FrameRequestResponse{
				Frame: frame,
			}
			log.Println("F:", f.IsFollow())
			log.Println("metadata:", string(f.Metadata()))
			log.Println("payload:", string(f.Payload()))
		case CANCEL:
			f := &FrameCancel{
				Frame: frame,
			}
			log.Println("F:", f.IsFollow())
			log.Println("C:", f.IsComplete())
			log.Println("N:", f.IsNext())
			log.Println("metadata:", string(f.Metadata()))
			log.Println("payload:", string(f.Payload()))
		case ERROR:
			f := &FrameError{
				Frame: frame,
			}
			log.Println("errorCode:", f.ErrorCode())
			log.Println("errorData:", string(f.ErrorData()))
		default:
			log.Println("frame:", frame)
		}
		log.Println("---------------------------------------")
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
		decoder: newLengthBasedFrameDecoder(c),
	}
}
