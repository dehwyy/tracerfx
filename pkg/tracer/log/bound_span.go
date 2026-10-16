package log

import "go.opentelemetry.io/otel/trace"

func boundSpan(l Logger) (Logger, trace.TraceID, trace.SpanID, bool) {
	switch bound := l.(type) {
	case *spanLogger:
		return bound.base, bound.traceID, bound.spanID, true
	case *lazySpanLogger:
		return bound.base, bound.traceID, bound.spanID, true
	}

	return l, trace.TraceID{}, trace.SpanID{}, false
}
