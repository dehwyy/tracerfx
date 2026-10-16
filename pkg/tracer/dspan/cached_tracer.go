package dspan

import (
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type tracerEntry struct {
	provider trace.TracerProvider
	tracer   trace.Tracer
}

var tracerCache atomic.Pointer[tracerEntry]

func sameProvider(left trace.TracerProvider, right trace.TracerProvider) (same bool) {
	defer func() {
		if recover() != nil {
			same = false
		}
	}()

	return left == right
}

func cachedTracer() trace.Tracer {
	provider := otel.GetTracerProvider()

	if entry := tracerCache.Load(); entry != nil && sameProvider(
		entry.provider,
		provider,
	) {
		return entry.tracer
	}

	entry := &tracerEntry{
		provider: provider,
		tracer:   provider.Tracer(""),
	}
	tracerCache.Store(entry)

	return entry.tracer
}
