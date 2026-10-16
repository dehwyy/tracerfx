package log

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

func LazyFromContext(ctx context.Context, sc trace.SpanContext) Logger {
	l := contextLogger(ctx)
	if !sc.IsValid() {
		return l
	}

	if _, traceID, spanID, bound := boundSpan(l); bound && traceID == sc.TraceID() && spanID == sc.SpanID() {
		return l
	}

	return NewLazySpanLogger(
		l,
		sc,
	)
}
