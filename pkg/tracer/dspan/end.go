package dspan

import (
	"time"
)

func (s *dspan) End() {
	s.TraceSpan.End()

	s.Mu.Lock()
	defer s.Mu.Unlock()

	duration := time.Since(s.StartTime)

	args := []any{
		"span_name", s.SpanName,
		"duration_ms", duration.Milliseconds(),
	}

	for k, v := range s.Attributes {
		args = append(args, k, v)
	}

	s.Logger.Debug("Span ended", args...)
}
