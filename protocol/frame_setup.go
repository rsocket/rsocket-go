package protocol

import "time"

type FrameSetup struct {
	Frame
}

func (p FrameSetup) IsResumeEnable() bool {
	return p.Frame.GetFlags().check(f7)
}

func (p FrameSetup) IsLease() bool {
	return p.Frame.GetFlags().check(f6)
}

func (p FrameSetup) GetMajor() uint16 {
	return byteOrder.Uint16(p.Frame[6:8])
}

func (p FrameSetup) GetMinor() uint16 {
	return byteOrder.Uint16(p.Frame[8:10])
}

func (p FrameSetup) GetTimeBetweenKeepalive() time.Duration {
	mills := int64(byteOrder.Uint32(p.Frame[10:14]))
	return time.Millisecond * time.Duration(mills)
}

func (p FrameSetup) GetMaxLifetime() time.Duration {
	mills := int64(byteOrder.Uint32(p.Frame[14:18]))
	return time.Millisecond * time.Duration(mills)
}

func (p FrameSetup) GetTokenLength() uint16 {
	if !p.IsResumeEnable() {
		return 0
	}
	return byteOrder.Uint16(p.Frame[18:20])
}

func (p FrameSetup) GetToken() uint16 {
	if !p.IsResumeEnable() {
		return 0
	}
	return byteOrder.Uint16(p.Frame[20:22])
}

func (p FrameSetup) GetMetadataMIME() []byte {
	offset, length := p.indexMetadataMIME()
	return p.Frame[offset+1 : offset+1+length]
}

func (p FrameSetup) GetDataMIME() []byte {
	offset, length := p.indexDataMIME()
	return p.Frame[offset+1 : offset+1+length]
}

func (p FrameSetup) GetPayload() []byte {
	o, l := p.indexDataMIME()
	offset := o + 1 + l
	return p.Frame[offset:]
}

func (p FrameSetup) indexMetadataMIME() (offset int, length int) {
	if p.IsResumeEnable() {
		offset = 22
	} else {
		offset = 18
	}
	length = int(p.Frame[offset])
	return
}

func (p FrameSetup) indexDataMIME() (offset int, length int) {
	o, l := p.indexMetadataMIME()
	offset = o + 1 + l
	length = int(p.Frame[offset])
	return
}
