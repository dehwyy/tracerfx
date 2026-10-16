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

	s.Mu.Lock()
	defer s.Mu.Unlock()

	if s.light {
		return s.errLight(
			err,
			expected,
		)
	}

	if expected {
		if !s.deferred {
			s.TraceSpan.SetAttributes(
				attribute.Bool(
					expectedErrorAttribute,
					true,
				),
			)
		}

		if s.fast && !log.DebugEnabled(s.Logger) {
			return err
		}

		s.Logger.Debug(
			"Span recorded expected error",
			s.errorArgs(err)...,
		)

		return err
	}

	s.expandPending()
	s.TraceSpan.SetStatus(
		codes.Error,
		err.Error(),
	)
	s.TraceSpan.RecordError(err)

	if s.scope != nil && s.scope.batch == s {
		s.scope.failed.Store(true)
	}

	s.Logger.Error(
		"Span recorded error",
		s.errorArgs(err)...,
	)

	return err
}
