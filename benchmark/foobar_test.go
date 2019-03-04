package benchmark

import (
	"context"
	"github.com/rsocket/rsocket-go"
	"strings"
	"testing"
)

func BenchmarkClient_RequestResponse(b *testing.B) {
	data := []byte(strings.Repeat("A", 4096))
	md := []byte("benchmark_test")
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cli := createClient("127.0.0.1", 8001)
			cli.RequestResponse(rsocket.NewPayload(data, md)).Subscribe(context.Background(), func(ctx context.Context, item rsocket.Payload) {
			})
			_ = cli.Close()
		}
	})
}
