package dspan

import (
	"context"
	"time"

	"github.com/dehwyy/tracerfx/pkg/tracer/caller"
	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"go.opentelemetry.io/otel"
)

func Start(ctx context.Context, spanName string, attrs ...Attribute) (context.Context, *dspan) {
	ctx, tSpan := otel.Tracer("").Start(ctx, spanName)
	logger := log.FromContext(ctx)

	s := &dspan{
		TraceSpan:  tSpan,
		Logger:     logger,
		Attributes: make(map[string]any),
		StartTime:  time.Now(),
		SpanName:   spanName,
		SpanCaller: caller.GetRuntimeFunc(2),
	}

	for _, attr := range attrs {
		s.WithAttribute(attr.Key, attr.Value)
	}

	logger.Info(
		"Span started",
		"span_name", spanName,
		"span_caller", s.SpanCaller,
		"trace_id", s.TraceID(),
	)

	return ctx, s
}
