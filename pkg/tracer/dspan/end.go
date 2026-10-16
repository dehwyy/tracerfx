package dspan

import (
	"time"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
)

func (s *dspan) End() {
	s.TraceSpan.End()

	if s.fast && !log.DebugEnabled(s.Logger) {
		return
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	duration := time.Since(s.StartTime)

	args := []any{
		"span_name", s.SpanName,
		"duration_ms", duration.Milliseconds(),
	}

	args = s.appendAttributeArgs(args)

	s.Logger.Debug("Span ended", args...)
}
