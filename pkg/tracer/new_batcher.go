package tracer

import (
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

func newBatcher(exporter tracesdk.SpanExporter, ratio float64) tracesdk.SpanProcessor {
	batcher := tracesdk.NewBatchSpanProcessor(exporter)
	if ratio >= 1 {
		return batcher
	}

	return newErrorPassProcessor(batcher)
}
