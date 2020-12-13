package mono

import (
	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

// Zip merges given Monos into a new Mono that will be fulfilled when all of the given Monos have produced an item, aggregating their values into a Tuple.
func Zip(first Mono, second Mono, others ...Mono) ZipBuilder {
	if len(others) < 1 {
		return []mono.Mono{
			unpackRawPublisher(first),
			unpackRawPublisher(second),
		}
	}
	sources := make([]mono.Mono, 2+len(others))
	sources[0] = unpackRawPublisher(first)
	sources[1] = unpackRawPublisher(second)
	for i := 0; i < len(others); i++ {
		sources[i+2] = unpackRawPublisher(others[i])
	}
	return sources
}

// ZipAll merges given Monos into a new Mono that will be fulfilled when all of the given Monos have produced an item, aggregating their values into a Tuple.
func ZipAll(sources ...Mono) ZipBuilder {
	all := make([]mono.Mono, len(sources))
	for i := 0; i < len(all); i++ {
		all[i] = unpackRawPublisher(sources[i])
	}
	return all
}

// ZipBuilder can be used to build a zipped Mono.
type ZipBuilder []mono.Mono

// ToMonoOneshot builds as a oneshot Mono.
func (z ZipBuilder) ToMonoOneshot(transform func(rx.Tuple) (payload.Payload, error)) Mono {
	return RawOneshot(mono.ZipCombineOneshot(cmb(transform), pinItem, z...))
}

// ToMono builds a Mono.
func (z ZipBuilder) ToMono(transform func(item rx.Tuple) (payload.Payload, error)) Mono {
	return Raw(mono.ZipCombine(cmb(transform), pinItem, z...))
}

func unpinItem(item *reactor.Item) {
	if item == nil {
		return
	}
	if r, _ := item.V.(common.Releasable); r != nil {
		r.Release()
	}
	if r, _ := item.E.(common.Releasable); r != nil {
		r.Release()
	}
}

func pinItem(item *reactor.Item) {
	if item == nil {
		return
	}
	if r, _ := item.V.(common.Releasable); r != nil {
		r.IncRef()
	}
	if r, _ := item.E.(common.Releasable); r != nil {
		r.IncRef()
	}
}

func cmb(transform func(rx.Tuple) (payload.Payload, error)) func(...*reactor.Item) (reactor.Any, error) {
	return func(values ...*reactor.Item) (reactor.Any, error) {
		defer func() {
			for i := 0; i < len(values); i++ {
				unpinItem(values[i])
			}
		}()
		t := rx.NewTuple(values...)
		return transform(t)
	}
}
