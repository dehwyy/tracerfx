package dspan

import "go.opentelemetry.io/otel/codes"

func (s *dspan) Err(err error) error {
	s.TraceSpan.SetStatus(codes.Error, err.Error())
	s.TraceSpan.RecordError(err)

	s.Mu.Lock()
	defer s.Mu.Unlock()

	args := []any{
		"span_name", s.SpanName,
		"trace_id", s.TraceID(),
		"error", err.Error(),
	}

	for k, v := range s.Attributes {
		args = append(args, k, v)
	}

	s.Logger.Error("Span recorded error", args...)

	return err
}
