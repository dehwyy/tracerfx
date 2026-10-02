package log

import "go.opentelemetry.io/otel/trace"

type spanLogger struct {
	Logger
	traceID trace.TraceID
	spanID  trace.SpanID
}

func newSpanLogger(base Logger, sc trace.SpanContext) Logger {
	if bound, ok := base.(*spanLogger); ok {
		base = bound.Logger
	}

	return &spanLogger{
		Logger: base.With(
			TraceIDKey, sc.TraceID().String(),
			SpanIDKey, sc.SpanID().String(),
		),
		traceID: sc.TraceID(),
		spanID:  sc.SpanID(),
	}
}

func (s *spanLogger) With(args ...any) Logger {
	return &spanLogger{
		Logger:  s.Logger.With(args...),
		traceID: s.traceID,
		spanID:  s.spanID,
	}
}
