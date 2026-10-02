package log

import "go.opentelemetry.io/otel/trace"

type spanLogger struct {
	Logger
	base    Logger
	traceID trace.TraceID
	spanID  trace.SpanID
}

func newSpanLogger(base Logger, sc trace.SpanContext) Logger {
	if bound, ok := base.(*spanLogger); ok {
		base = bound.base
	}

	return bindSpan(base, sc.TraceID(), sc.SpanID())
}

func bindSpan(base Logger, traceID trace.TraceID, spanID trace.SpanID) *spanLogger {
	return &spanLogger{
		Logger: base.With(
			TraceIDKey, traceID.String(),
			SpanIDKey, spanID.String(),
		),
		base:    base,
		traceID: traceID,
		spanID:  spanID,
	}
}

func (s *spanLogger) With(args ...any) Logger {
	return bindSpan(s.base.With(args...), s.traceID, s.spanID)
}
