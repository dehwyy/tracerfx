package log

func (l *lazySpanLogger) Debug(msg string, args ...any) {
	l.resolve().Debug(
		msg,
		args...,
	)
}
