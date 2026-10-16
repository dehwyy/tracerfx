package dspan

import (
	"context"
	"time"

	"github.com/dehwyy/tracerfx/pkg/tracer/caller"
	"github.com/dehwyy/tracerfx/pkg/tracer/log"
)

func beginFast(ctx context.Context, spanName string, attrs []Attribute, callerSkip int) (context.Context, *dspan) {
	ctx, tSpan := cachedTracer().Start(
		ctx,
		spanName,
	)
	spanContext := tSpan.SpanContext()
	logger := log.LazyFromContext(
		ctx,
		spanContext,
	)

	s := &dspan{
		TraceSpan: tSpan,
		Logger:    logger,
		StartTime: time.Now(),
		SpanName:  spanName,
		fast:      true,
		deferred:  !spanContext.IsSampled(),
	}

	if !s.deferred {
		s.Attributes = make(map[string]any)
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

	return ctx, s
}
