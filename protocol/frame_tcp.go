package protocol

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
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
		if len(data) >= 3 {
			frameLength := readUint24(data)
			if frameLength < 1 {
				err = fmt.Errorf("bad frame length: %d", frameLength)
				return
			}
			log.Println("frame length:", frameLength)
			frameSize := frameLength + 3
			if frameSize <= len(data) {
				return frameSize, data[:frameSize], nil
			}
		}
		return
	})
	for p.scanner.Scan() {
		data := p.scanner.Bytes()
		//log.Println("scan:", data)
		f, err := bytes2frame(data[3:])
		if err != nil {
			return err
		}
		if err := fn(f); err != nil {
			return err
		}
	}
	return p.scanner.Err()
}

func bytes2frame(bs []byte) (*Frame, error) {
	streamID := byteOrder.Uint32(bs[:4])
	foo := byteOrder.Uint16(bs[4:6])
	frameType := FrameType((foo & 0xFC00) >> 10)
	isIgnore := 0x0200&foo == 0x0200
	isMetadata := 0x0100&foo == 0x0100
	frameFlags := uint8(0x00FF & foo)
	frameHeader := FrameHeader{
		StreamID:   streamID,
		Type:       frameType,
		IsIgnore:   isIgnore,
		IsMetadata: isMetadata,
		Flags:      frameFlags,
	}

	bs = bs[6:]

	switch frameType {
	case SETUP:
		log.Println("-------------< SETUP >-------------")
		isResumeEnable := 0x80&frameFlags == 0x80
		log.Println("R:", isResumeEnable)
		isLease := 0x40&frameFlags == 0x40
		log.Println("L:", isLease)
		majorVersion := byteOrder.Uint16(bs[:2])
		log.Println("major:", majorVersion)
		minorVersion := byteOrder.Uint16(bs[2:4])
		log.Println("minor:", minorVersion)
		timeBetweenKeepalive := byteOrder.Uint32(bs[4:8])
		log.Println("timeBetweenKeepalive:", timeBetweenKeepalive)
		maxLifetime := byteOrder.Uint32(bs[8:12])
		log.Println("maxLifetime:", maxLifetime)

		bs = bs[12:]
		if isResumeEnable {
			tokenLength := byteOrder.Uint16(bs[12:14])
			log.Println("tokenLength:", tokenLength)
			token := byteOrder.Uint16(bs[14:16])
			log.Println("token:", token)
			bs = bs[16:]
		}

		l1 := int(bs[0])
		m1 := bs[1 : 1+l1]
		log.Println("m1:", string(m1))
		bs = bs[1+l1:]

		l2 := int(bs[0])
		m2 := bs[1 : 1+l2]
		log.Println("m2:", string(m2))

		bs = bs[1+l2:]

		ml := readUint24(bs)

		if ml > 0 {
			md := bs[3 : 3+ml]
			log.Println("metadata:", md)
		} else {
			log.Println("metadata:", nil)
		}
		payload := bs[3+ml:]
		log.Println("payload:", string(payload))
		log.Println("-----------------------------------")
	}

	//if isMetadata {
	//	metadataLength := readUint24WithOffset(bs, 7)
	//	log.Println("read metadata length:", metadataLength)
	//}
	return &Frame{
		Header: frameHeader,
	}, nil
}

func newTcpFrameDecoder(c net.Conn) *tcpFrameDecoder {
	return &tcpFrameDecoder{
		scanner: bufio.NewScanner(c),
	}
}
