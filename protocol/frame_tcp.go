package protocol

import (
	"bufio"
	"log"
	"net"
)

type tcpFrameDecoder struct {
	r           *bufio.Reader
	frameLength int
}

func (p *tcpFrameDecoder) sliceFrameLength() error {
	bf := make([]byte, 3)

	var read int

	for read < 3 {
		n, err := p.r.Read(bf)
		if err != nil {
			log.Println("err:", err)
			break
		}
		read += n
		log.Printf("read=%d,n=%d,b=0x%x0x%x0x%x\n", read, n, bf[0], bf[1], bf[2])
	}

	log.Println("read:", bf[0], bf[1], bf[2])
	p.frameLength = int(bf[0])<<16 + int(bf[1])<<8 + int(bf[2])
	log.Println("read frame length:", p.frameLength)
	return nil
}

func newTcpFrameDecoder(c net.Conn) *tcpFrameDecoder {
	return &tcpFrameDecoder{
		r: bufio.NewReader(c),
	}
}
