package log

func (l *lazySpanLogger) Info(msg string, args ...any) {
	l.resolve().Info(
		msg,
		args...,
	)
}
