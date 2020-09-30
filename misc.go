package rsocket

import (
	"log"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/socket"
)

func TracePoolCount() {
	log.Printf("*** trace count: bytebuff=%d, requests=%d ***\n", common.CountByteBuff(), socket.CountRequestResponseRef())
}
