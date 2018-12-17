package rsocket

type Fragmenter struct {
	store map[uint32]*frameHolder
}

type frameHolder struct {
	header      *Header
	bufMetadata []byte
	bufData     []byte
}

func (p *frameHolder) append(f *FramePayload) {
	p.header = f.Header
	m := f.Metadata()
	if m != nil && len(m) > 0 {
		p.bufMetadata = append(p.bufMetadata, m...)
	}
	d := f.Data()
	if d != nil && len(d) > 0 {
		p.bufData = append(p.bufData, d...)
	}
}

func (p *Fragmenter) Fragment(f *FramePayload) *FramePayload {
	isFollow := f.Header.Flags().Check(FlagFollow)
	sid := f.Header.StreamID()
	found, ok := p.store[sid]
	if isFollow {
		if ok {
			found.append(f)
		} else {
			found = &frameHolder{
				header:      f.Header,
				bufData:     f.Data(),
				bufMetadata: f.Metadata(),
			}
			p.store[sid] = found
		}
		return nil
	}
	if !ok {
		return f
	}
	found.append(f)
	ret := mkPayload(sid, found.bufMetadata, found.bufData, found.header.Flags())
	delete(p.store, sid)
	return ret
}
