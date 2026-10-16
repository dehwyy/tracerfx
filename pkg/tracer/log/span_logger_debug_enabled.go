package log

func (s *spanLogger) DebugEnabled() bool {
	return DebugEnabled(s.base)
}
