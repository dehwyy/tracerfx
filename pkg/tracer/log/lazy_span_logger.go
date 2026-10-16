package log

import (
	"sync"

	"go.opentelemetry.io/otel/trace"
)

type lazySpanLogger struct {
	base    Logger
	traceID trace.TraceID
	spanID  trace.SpanID
	once    sync.Once
	bound   *spanLogger
}

func NewLazySpanLogger(base Logger, sc trace.SpanContext) Logger {
	unwrapped, _, _, _ := boundSpan(base)

	return &lazySpanLogger{
		base:    unwrapped,
		traceID: sc.TraceID(),
		spanID:  sc.SpanID(),
	}
}

func (l *lazySpanLogger) resolve() *spanLogger {
	l.once.Do(
		func() {
			l.bound = bindSpan(
				l.base,
				l.traceID,
				l.spanID,
			)
		},
	)

	return l.bound
}
