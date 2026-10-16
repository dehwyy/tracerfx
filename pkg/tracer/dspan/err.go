package dspan

import (
	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func (s *dspan) Err(err error) error {
	if err == nil {
		return nil
	}

	expected := isExpectedError(err)

	if expected {
		s.TraceSpan.SetAttributes(
			attribute.Bool(
				expectedErrorAttribute,
				true,
			),
		)
	} else {
		s.TraceSpan.SetStatus(
			codes.Error,
			err.Error(),
		)
		s.TraceSpan.RecordError(err)
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	args := []any{
		"span_name", s.SpanName,
		log.Err(err),
	}

	for k, v := range s.Attributes {
		args = append(
			args,
			k,
			v,
		)
	}

	if expected {
		s.Logger.Debug(
			"Span recorded expected error",
			args...,
		)

		return err
	}

	s.Logger.Error(
		"Span recorded error",
		args...,
	)

	return err
}
