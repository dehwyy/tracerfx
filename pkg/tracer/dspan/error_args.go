package dspan

import (
	"github.com/dehwyy/tracerfx/pkg/tracer/log"
)

func (s *dspan) errorArgs(err error) []any {
	args := []any{
		"span_name", s.SpanName,
		log.Err(err),
	}

	return s.appendAttributeArgs(args)
}
