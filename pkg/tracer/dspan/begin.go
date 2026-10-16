package dspan

import (
	"context"
	"time"

	"github.com/dehwyy/tracerfx/pkg/tracer/caller"
	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"go.opentelemetry.io/otel"
)

const callerSkipStart = 3

func begin(ctx context.Context, spanName string, attrs []Attribute, callerSkip int) (context.Context, *dspan) {
	if depthSuppresses(
		spanDepth(),
		spanName,
	) {
		if scope := activeScope(ctx); scope != nil {
			return ctx, startLight(
				ctx,
				scope,
				spanName,
				attrs,
				callerSkip,
			)
		}
	}

	if fastEnabled() {
		return beginFast(
			ctx,
			spanName,
			attrs,
			callerSkip,
		)
	}

	ctx, tSpan := otel.Tracer("").Start(
		ctx,
		spanName,
	)
	logger := log.FromContext(ctx)

	s := &dspan{
		TraceSpan:  tSpan,
		Logger:     logger,
		Attributes: make(map[string]any),
		StartTime:  time.Now(),
		SpanName:   spanName,
		SpanCaller: caller.GetRuntimeFunc(callerSkip),
	}

	for _, attr := range attrs {
		s.WithAttribute(
			attr.Key,
			attr.Value,
		)
	}

	logger.Debug(
		"Span started",
		"span_name",
		spanName,
		"span_caller",
		s.SpanCaller,
	)

	return ctx, s
}
