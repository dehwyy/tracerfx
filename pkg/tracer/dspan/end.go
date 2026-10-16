package dspan

import (
	"time"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"go.opentelemetry.io/otel/attribute"
)

func (s *dspan) End() {
	if s.scope != nil && s.scope.batch == s {
		s.TraceSpan.SetAttributes(
			attribute.Bool(
				batchFailedAttribute,
				s.scope.failed.Load(),
			),
			attribute.Int64(
				batchSuppressedAttribute,
				s.scope.suppressed.Load(),
			),
		)
	}

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
