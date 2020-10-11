package fragmentation

import (
	"container/list"
	"fmt"

	"github.com/rsocket/rsocket-go/core"
)

type implJoiner struct {
	root *list.List // list of HeaderAndPayload
}

func (p *implJoiner) IncRef() (refs int32) {
	for cur := p.root.Front(); cur != nil; cur = cur.Next() {
		if r, ok := cur.Value.(core.BufferedFrame); ok {
			refs = r.IncRef()
		}
	}
	return
}

func (p *implJoiner) RefCnt() int32 {
	return p.root.Front().Value.(core.BufferedFrame).RefCnt()
}

func (p *implJoiner) Release() {
	for cur := p.root.Front(); cur != nil; cur = cur.Next() {
		if r, ok := cur.Value.(core.BufferedFrame); ok {
			r.Release()
		}
	}
}

func (p *implJoiner) First() core.BufferedFrame {
	first := p.root.Front()
	if first == nil {
		panic("no frames in current joiner")
	}
	return first.Value.(core.BufferedFrame)
}

func (p *implJoiner) Header() core.FrameHeader {
	return p.First().Header()
}

func (p *implJoiner) String() string {
	m, _ := p.MetadataUTF8()
	return fmt.Sprintf("Joiner{data=%s,metadata=%s}", p.DataUTF8(), m)
}

func (p *implJoiner) Metadata() (metadata []byte, ok bool) {
	for cur := p.root.Front(); cur != nil; cur = cur.Next() {
		f := cur.Value.(HeaderAndPayload)
		if !f.Header().Flag().Check(core.FlagMetadata) {
			break
		}
		if m, has := f.Metadata(); has {
			metadata = append(metadata, m...)
			ok = true
		}
	}
	return
}

func (p *implJoiner) MetadataUTF8() (metadata string, ok bool) {
	var m []byte
	m, ok = p.Metadata()
	if ok {
		metadata = string(m)
	}
	return
}

func (p *implJoiner) Data() (data []byte) {
	for cur := p.root.Front(); cur != nil; cur = cur.Next() {
		f := cur.Value.(HeaderAndPayload)
		if d := f.Data(); len(d) > 0 {
			data = append(data, d...)
		}
	}
	return
}

func (p *implJoiner) DataUTF8() (data string) {
	if d := p.Data(); len(d) > 0 {
		data = string(d)
	}
	return
}

func (p *implJoiner) Push(elem HeaderAndPayload) (end bool) {
	p.root.PushBack(elem)
	h := elem.Header()
	end = !h.Flag().Check(core.FlagFollow)
	return
}
