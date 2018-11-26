package protocol

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
)

var (
	byteOrder = binary.BigEndian
)

type tcpFrameDecoder struct {
	scanner *bufio.Scanner
}

func (p *tcpFrameDecoder) Handle(fn FrameHandler) error {
	p.scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF {
			return
		}
		if len(data) < 3 {
			return
		}
		frameLength := readUint24(data)
		if frameLength < 1 {
			err = fmt.Errorf("bad frame length: %d", frameLength)
			return
		}
		frameSize := frameLength + 3
		if frameSize <= len(data) {
			return frameSize, data[:frameSize], nil
		}
		return
	})
	for p.scanner.Scan() {
		data := p.scanner.Bytes()
		f, err := FromBytes(data[3:])
		if err != nil {
			return err
		}
		if err := fn(f); err != nil {
			return err
		}
	}
	return p.scanner.Err()
}

func newTcpFrameDecoder(c net.Conn) *tcpFrameDecoder {
	return &tcpFrameDecoder{
		scanner: bufio.NewScanner(c),
	}
}
