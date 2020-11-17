package mono

import (
	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/jjeffcaii/reactor-go/tuple"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

func Zip(first Mono, second Mono, others ...Mono) ZipBuilder {
	var all []Mono
	all = append(all, first, second)
	all = append(all, others...)
	return ZipAll(all...)
}

func ZipAll(sources ...Mono) ZipBuilder {
	if len(sources) < 1 {
		panic("at least one Mono for zip operation")
	}
	all := make([]mono.Mono, len(sources))
	for i := 0; i < len(all); i++ {
		all[i] = sources[i].Raw()
	}
	return all
}

type ZipBuilder []mono.Mono

func (z ZipBuilder) ToMono(transform func(rx.Tuple) (payload.Payload, error)) Mono {
	return Raw(mono.ZipAll(z...).Map(func(any reactor.Any) (reactor.Any, error) {
		tup := rx.NewTuple(any.(tuple.Tuple))
		return transform(tup)
	}))
}
