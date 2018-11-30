package rsocket

type FrameCancel struct {
	*Header
}

func asCancel(h *Header, raw []byte) *FrameCancel {
	return &FrameCancel{
		Header: h,
	}
}

func mkCancel(sid uint32) *FrameCancel {
	return &FrameCancel{
		Header: mkHeader(sid, CANCEL),
	}
}
