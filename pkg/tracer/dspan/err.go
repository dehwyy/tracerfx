package dspan

import (
	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"go.opentelemetry.io/otel/codes"
)

func (s *dspan) Err(err error) error {
	if err == nil {
		return nil
	}

	s.TraceSpan.SetStatus(codes.Error, err.Error())
	s.TraceSpan.RecordError(err)

	s.Mu.Lock()
	defer s.Mu.Unlock()

	args := []any{
		"span_name", s.SpanName,
		log.Err(err),
	}

	for k, v := range s.Attributes {
		args = append(args, k, v)
	}

	s.Logger.Error("Span recorded error", args...)

	return err
}
