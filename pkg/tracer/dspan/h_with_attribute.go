package dspan

func (s *dspan) WithAttribute(key string, value any) *dspan {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	extracted := extractFields(key, value)

	for k, v := range extracted {
		s.Attributes[k] = v
		setAttr(s.TraceSpan, k, v)
	}

	return s
}
