package dspan

import (
	"context"
	"time"

	"github.com/dehwyy/tracerfx/pkg/tracer/caller"
	"github.com/dehwyy/tracerfx/pkg/tracer/log"
)

func startLight(ctx context.Context, scope *batchScope, spanName string, attrs []Attribute, callerSkip int) *dspan {
	scope.suppressed.Add(1)

	logger := log.LazyFromContext(
		ctx,
		scope.spanContext,
	)

	s := &dspan{
		TraceSpan: scope.nonRecording,
		Logger:    logger,
		StartTime: time.Now(),
		SpanName:  spanName,
		fast:      true,
		deferred:  true,
		light:     true,
		scope:     scope,
	}

	for _, attr := range attrs {
		s.WithAttribute(
			attr.Key,
			attr.Value,
		)
	}

	if log.DebugEnabled(logger) {
		s.SpanCaller = caller.GetRuntimeFunc(callerSkip + 1)

		logger.Debug(
			"Span started",
			"span_name",
			spanName,
			"span_caller",
			s.SpanCaller,
		)
	}

	return s
}
