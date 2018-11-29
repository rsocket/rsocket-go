package rsocket

type HandlerRQ = func(request *Payload) (reponse *Payload, err error)
