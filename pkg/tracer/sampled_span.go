package tracer

import (
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type sampledSpan struct {
	tracesdk.ReadOnlySpan
}

func (s sampledSpan) SpanContext() trace.SpanContext {
	spanContext := s.ReadOnlySpan.SpanContext()

	return spanContext.WithTraceFlags(spanContext.TraceFlags() | trace.FlagsSampled)
}
