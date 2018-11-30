package rsocket

type Client struct {
	transport Transport
	acceptor  Acceptor
	handlerRQ HandlerRQ
	handlerRS HandlerRS
}


