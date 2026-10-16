package tracer

import (
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

type errorPassProcessor struct {
	tracesdk.SpanProcessor
}

func newErrorPassProcessor(next tracesdk.SpanProcessor) tracesdk.SpanProcessor {
	return errorPassProcessor{
		SpanProcessor: next,
	}
}

func (p errorPassProcessor) OnEnd(span tracesdk.ReadOnlySpan) {
	if span.SpanContext().IsSampled() {
		p.SpanProcessor.OnEnd(span)
		return
	}

	if span.Status().Code != codes.Error {
		return
	}

	p.SpanProcessor.OnEnd(
		sampledSpan{
			ReadOnlySpan: span,
		},
	)
}
