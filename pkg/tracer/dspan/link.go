package dspan

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

const maxLinks = 128

func (s *dspan) Link(ctx context.Context) {
	if ctx == nil {
		return
	}

	linked := trace.SpanContextFromContext(ctx)
	if !linked.IsValid() {
		return
	}

	target := s
	if s.light && s.scope != nil {
		target = s.scope.batch
	}

	if linked.SpanID() == target.TraceSpan.SpanContext().SpanID() {
		return
	}

	if !target.reserveLink() {
		return
	}

	target.TraceSpan.AddLink(
		trace.Link{
			SpanContext: linked,
		},
	)
}
