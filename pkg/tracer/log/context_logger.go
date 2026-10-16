package log

import "context"

func contextLogger(ctx context.Context) Logger {
	if ctx == nil {
		return Default()
	}

	l, ok := ctx.Value(loggerContextKey{}).(Logger)
	if !ok || l == nil {
		return Default()
	}

	return l
}
