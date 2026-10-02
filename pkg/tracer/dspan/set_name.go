package dspan

func (s *dspan) SetName(name string) {
	s.TraceSpan.SetName(name)

	s.Mu.Lock()
	defer s.Mu.Unlock()

	s.SpanName = name
}
