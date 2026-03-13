package dspan

func (s *dspan) TraceID() string {
	return s.TraceSpan.SpanContext().TraceID().String()
}
