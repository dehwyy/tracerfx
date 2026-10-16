package log

type DebugEnabler interface {
	DebugEnabled() bool
}

func DebugEnabled(l Logger) bool {
	if enabler, ok := l.(DebugEnabler); ok {
		return enabler.DebugEnabled()
	}

	return true
}
