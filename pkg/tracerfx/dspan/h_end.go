package dspan

import "time"

func (s *dspan) End() {
	s.TraceSpan.End()

	s.Mu.Lock()
	defer s.Mu.Unlock()

	duration := time.Since(s.StartTime)

	args := []any{
		"span_name", s.SpanName,
		"trace_id", s.TraceID(),
		"duration_ms", duration.Milliseconds(),
	}

	for k, v := range s.Attributes {
		args = append(args, k, v)
	}

	s.Logger.Info("Span ended", args...)
}
