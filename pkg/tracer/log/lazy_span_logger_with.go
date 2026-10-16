package log

func (l *lazySpanLogger) With(args ...any) Logger {
	return l.resolve().With(args...)
}
