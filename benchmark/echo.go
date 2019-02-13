package benchmark

import (
	"fmt"
	"github.com/rsocket/rsocket-go"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strconv"
)

func runEchoServer(host string, port int) *rsocket.Server {
	go func() {
		log.Println(http.ListenAndServe(":4444", nil))
	}()
	server, err := rsocket.NewServer(
		rsocket.WithTCPServerTransport(fmt.Sprintf("%s:%d", host, port)),
		rsocket.WithAcceptor(func(setup rsocket.SetupPayload, rs *rsocket.RSocket) (err error) {
			log.Println("SETUP BEGIN:----------------")
			log.Println("maxLifeTime:", setup.MaxLifetime())
			log.Println("keepaliveInterval:", setup.TimeBetweenKeepalive())
			log.Println("dataMimeType:", setup.DataMimeType())
			log.Println("metadataMimeType:", setup.MetadataMimeType())
			log.Println("data:", string(setup.Data()))
			log.Println("metadata:", string(setup.Metadata()))
			log.Println("SETUP END:----------------")

			rs.HandleMetadataPush(func(metadata []byte) {
				log.Println("GOT METADATA_PUSH:", string(metadata))
			})

			rs.HandleFireAndForget(func(req rsocket.Payload) {
				log.Println("GOT FNF:", req)
			})

			rs.HandleRequestResponse(func(req rsocket.Payload) (res rsocket.Payload, err error) {
				// just echo
				return req, nil
			})

			rs.HandleRequestStream(func(req rsocket.Payload) rsocket.Flux {
				d := string(req.Data())
				totals, _ := strconv.Atoi(string(req.Metadata()))
				return rsocket.NewFlux(func(emitter rsocket.Emitter) {
					for i := 0; i < totals; i++ {
						payload := rsocket.NewPayload([]byte(fmt.Sprintf("%s_%d", d, i)), nil)
						emitter.Next(payload)
					}
					emitter.Complete()
				})
			})

			rs.HandleRequestChannel(func(req rsocket.Flux) (res rsocket.Flux) {
				return req
				//// echo all incoming payloads
				//f := rsocket.NewFlux(func(emitter rsocket.Emitter) {
				//	req.
				//		DoFinally(func() {
				//			emitter.Complete()
				//		}).
				//		SubscribeOn(rsocket.ElasticScheduler()).
				//		Subscribe(func(item rsocket.Payload) {
				//			emitter.Next(rsocket.NewPayload(item.Data(), item.Metadata()))
				//		})
				//})
				//return f
			})
			return nil
		}),
	)
	if err != nil {
		panic(err)
	}
	return server
}
