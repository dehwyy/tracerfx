package dspan

import (
	"context"
	"time"

	"github.com/dehwyy/tracerfx/pkg/tracerfx/caller"
	"github.com/dehwyy/tracerfx/pkg/tracerfx/log"
	"go.opentelemetry.io/otel"
)

func Start(ctx context.Context) (context.Context, *dspan) {
	spanName := caller.GetRuntimeFunc(2)
	ctx, tSpan := otel.Tracer("").Start(ctx, spanName)
	logger := log.FromContext(ctx)

	s := &dspan{
		TraceSpan:  tSpan,
		Logger:     logger,
		Attributes: make(map[string]any),
		StartTime:  time.Now(),
		SpanName:   spanName,
	}

	logger.Info(
		"Span started",
		"span_name", spanName,
		"trace_id", s.TraceID(),
	)

	return ctx, s
}
