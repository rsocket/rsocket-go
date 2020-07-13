package transport_test

import (
	"bytes"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/internal/common"
)

type mockConn struct {
	spy map[string]int
	c   chan core.FrameSupport
}

func (m *mockConn) call(fn string) {
	m.spy[fn] = m.spy[fn] + 1
}

func (m *mockConn) Close() error {
	m.call("Close")
	return nil
}

func (m *mockConn) SetDeadline(deadline time.Time) error {
	m.call("SetDeadline")
	return nil
}

func (m *mockConn) SetCounter(c *core.Counter) {
	m.call("SetCounter")
}

func (m *mockConn) Read() (next core.Frame, err error) {
	f := <-m.c
	bf := &bytes.Buffer{}
	_, err = f.WriteTo(bf)
	if err != nil {
		return
	}
	bs := bf.Bytes()
	header := core.ParseFrameHeader(bs)
	bb := common.NewByteBuff()
	_, err = bb.Write(bs[core.FrameHeaderLen:])
	if err != nil {
		return
	}
	next, err = framing.FromRawFrame(framing.NewRawFrame(header, bb))
	return
}

func (m *mockConn) Write(support core.FrameSupport) (err error) {
	m.c <- support
	return
}

func (m *mockConn) Flush() (err error) {
	m.call("Flush")
	return
}
