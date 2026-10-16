package dspan

import (
	"github.com/dehwyy/tracerfx/pkg/tracer/log"
)

func (s *dspan) errLight(err error, expected bool) error {
	if expected {
		if log.DebugEnabled(s.Logger) {
			s.Logger.Debug(
				"Span recorded expected error",
				s.errorArgs(err)...,
			)
		}

		return err
	}

	s.Logger.Error(
		"Span recorded error",
		s.errorArgs(err)...,
	)

	s.scope.recordFailure(err)

	return err
}
