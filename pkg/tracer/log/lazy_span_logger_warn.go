package log

func (l *lazySpanLogger) Warn(msg string, args ...any) {
	l.resolve().Warn(
		msg,
		args...,
	)
}
