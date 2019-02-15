package rsocket

type RTransport interface {
	HandleSetup(callback func(setup *frameSetup) (err error))
	HandleFNF(callback func(f *frameFNF) (err error))
	HandleMetadataPush(callback func(f *frameMetadataPush) (err error))
	HandleRequestResponse(callback func(f *frameRequestResponse) (err error))
	HandleRequestStream(callback func(f *frameRequestStream) (err error))
	HandleRequestChannel(callback func(f *frameRequestChannel) (err error))
	HandlePayload(callback func(f *framePayload) (err error))
	HandleRequestN(callback func(f *frameRequestN) (err error))
}
