package log

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/otel/trace"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

type DebugLogger = Logger

func Debug(l Logger, msg string, args ...any) {
	l.Debug(msg, args...)
}

type Attr struct {
	Key   string
	Value any
}

func Err(err error) Attr {
	return Attr{
		Key:   ErrorKey,
		Value: err,
	}
}

const (
	ErrorKey   = "error"
	TraceIDKey = "trace_id"
	SpanIDKey  = "span_id"
	BadKey     = "!BADKEY"
)

var defaultLogger atomic.Pointer[Logger]

var fallbackLogger atomic.Pointer[Logger]

func SetDefaultLogger(l Logger) {
	if l == nil {
		defaultLogger.Store(nil)
		return
	}
	defaultLogger.Store(&l)
}

func Default() Logger {
	if l := defaultLogger.Load(); l != nil {
		return *l
	}
	if l := fallbackLogger.Load(); l != nil {
		return *l
	}
	l := NewZerologLogger()
	fallbackLogger.CompareAndSwap(nil, &l)
	return *fallbackLogger.Load()
}

type loggerContextKey struct{}

func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

func FromContext(ctx context.Context) Logger {
	l := contextLogger(ctx)
	if ctx == nil {
		return l
	}

	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return l
	}

	if _, traceID, spanID, bound := boundSpan(l); bound && traceID == sc.TraceID() && spanID == sc.SpanID() {
		return l
	}

	return newSpanLogger(l, sc)
}
