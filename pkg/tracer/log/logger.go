package log

import "context"

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

type DebugLogger interface {
	Debug(msg string, args ...any)
}

func Debug(l Logger, msg string, args ...any) {
	if d, ok := l.(DebugLogger); ok {
		d.Debug(msg, args...)
	}
}

var defaultLogger Logger

func SetDefaultLogger(l Logger) {
	defaultLogger = l
}

type loggerContextKey struct{}

func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(loggerContextKey{}).(Logger); ok && l != nil {
		return l
	}
	if defaultLogger != nil {
		return defaultLogger
	}
	return NewZerologLogger()
}
