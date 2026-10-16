package dspan

import (
	"context"
	"io"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"go.opentelemetry.io/otel"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type benchRequest struct {
	OrderID string
	Amount  int64
	Status  string
	Nested  fastInner
}

func benchProvider(b *testing.B, sampler tracesdk.Sampler) {
	b.Helper()

	previous := otel.GetTracerProvider()
	provider := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(sampler),
		tracesdk.WithSpanProcessor(
			errorPassProbe{
				SpanProcessor: tracesdk.NewSimpleSpanProcessor(tracetest.NewInMemoryExporter()),
			},
		),
	)
	otel.SetTracerProvider(provider)

	b.Cleanup(
		func() {
			otel.SetTracerProvider(previous)
		},
	)
}

func benchLifecycle(b *testing.B, ctx context.Context) {
	b.Helper()

	request := &benchRequest{
		OrderID: "order-1",
		Amount:  100,
		Status:  "NEW",
		Nested: fastInner{
			Value: 1,
		},
	}
	response := &benchRequest{
		OrderID: "order-1",
		Amount:  100,
		Status:  "DONE",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, span := Start(
			ctx,
			"application.service.Method",
			Attr(
				"req",
				request,
			),
		)
		Response(
			span,
			response,
		)
		span.End()
	}
}

func BenchmarkSpanLifecycleUnsampled(b *testing.B) {
	modes := []struct {
		name string
		fast bool
	}{
		{
			name: "off",
			fast: false,
		},
		{
			name: "on",
			fast: true,
		},
	}

	for _, mode := range modes {
		b.Run(
			mode.name,
			func(b *testing.B) {
				state := fastStateOff
				if mode.fast {
					state = fastStateOn
				}
				fastState.Store(state)
				b.Cleanup(
					func() {
						fastState.Store(fastStateUnset)
					},
				)

				benchProvider(
					b,
					recordOnlyTestSampler{},
				)

				ctx := log.ContextWithLogger(
					context.Background(),
					log.New(
						log.WithWriter(io.Discard),
						log.WithLevel(log.LevelInfo),
					),
				)

				benchLifecycle(
					b,
					ctx,
				)
			},
		)
	}
}

func BenchmarkSpanLifecycleNeverSample(b *testing.B) {
	modes := []struct {
		name string
		fast bool
	}{
		{
			name: "off",
			fast: false,
		},
		{
			name: "on",
			fast: true,
		},
	}

	for _, mode := range modes {
		b.Run(
			mode.name,
			func(b *testing.B) {
				state := fastStateOff
				if mode.fast {
					state = fastStateOn
				}
				fastState.Store(state)
				b.Cleanup(
					func() {
						fastState.Store(fastStateUnset)
					},
				)

				benchProvider(
					b,
					tracesdk.NeverSample(),
				)

				ctx := log.ContextWithLogger(
					context.Background(),
					log.New(
						log.WithWriter(io.Discard),
						log.WithLevel(log.LevelInfo),
					),
				)

				benchLifecycle(
					b,
					ctx,
				)
			},
		)
	}
}

func BenchmarkSpanLifecycleSampled(b *testing.B) {
	modes := []struct {
		name string
		fast bool
	}{
		{
			name: "off",
			fast: false,
		},
		{
			name: "on",
			fast: true,
		},
	}

	for _, mode := range modes {
		b.Run(
			mode.name,
			func(b *testing.B) {
				state := fastStateOff
				if mode.fast {
					state = fastStateOn
				}
				fastState.Store(state)
				b.Cleanup(
					func() {
						fastState.Store(fastStateUnset)
					},
				)

				benchProvider(
					b,
					tracesdk.AlwaysSample(),
				)

				ctx := log.ContextWithLogger(
					context.Background(),
					log.New(
						log.WithWriter(io.Discard),
						log.WithLevel(log.LevelInfo),
					),
				)

				benchLifecycle(
					b,
					ctx,
				)
			},
		)
	}
}
