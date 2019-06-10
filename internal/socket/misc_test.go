package socket

import (
	"math"
	"testing"
	"time"
)

func BenchmarkMessageStore(b *testing.B) {
	store := newMailboxes()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := uint32(time.Now().UnixNano() % math.MaxUint32)
			store.put(id, mailRequestResponse, nil, nil)
			store.load(id)
			store.remove(id)
		}
	})
}

func BenchmarkPublishersPool(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			v := borrowPublishers(mailRequestResponse, nil, nil)
			returnPublishers(v)
		}
	})
}
