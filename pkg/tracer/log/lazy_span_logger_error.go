package log

func (l *lazySpanLogger) Error(msg string, args ...any) {
	l.resolve().Error(
		msg,
		args...,
	)
}
