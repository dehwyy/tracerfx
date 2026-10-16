package dspan

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/otel/trace"
)

type batchScopeKey struct{}

type batchScope struct {
	batch        *dspan
	spanContext  trace.SpanContext
	nonRecording trace.Span
	suppressed   atomic.Int64
	failed       atomic.Bool
}

func newBatchScope(batch *dspan) *batchScope {
	spanContext := batch.TraceSpan.SpanContext()

	return &batchScope{
		batch:       batch,
		spanContext: spanContext,
		nonRecording: trace.SpanFromContext(
			trace.ContextWithSpanContext(
				context.Background(),
				spanContext,
			),
		),
	}
}

func activeScope(ctx context.Context) *batchScope {
	if ctx == nil {
		return nil
	}

	scope, ok := ctx.Value(batchScopeKey{}).(*batchScope)
	if !ok || scope == nil {
		return nil
	}

	if trace.SpanContextFromContext(ctx).TraceID() != scope.spanContext.TraceID() {
		return nil
	}

	return scope
}
