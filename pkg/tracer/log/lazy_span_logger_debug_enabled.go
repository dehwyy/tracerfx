package log

func (l *lazySpanLogger) DebugEnabled() bool {
	return DebugEnabled(l.base)
}
