package dspan

import (
	"context"
)

func StartBatch(ctx context.Context, spanName string, attrs ...Attribute) (context.Context, *dspan) {
	if spanDepth() == depthAll || activeScope(ctx) != nil {
		return begin(
			ctx,
			spanName,
			attrs,
			callerSkipStart,
		)
	}

	ctx, batch := begin(
		ctx,
		spanName,
		attrs,
		callerSkipStart,
	)

	if !batch.TraceSpan.SpanContext().IsValid() {
		return ctx, batch
	}

	scope := newBatchScope(batch)
	batch.scope = scope

	return context.WithValue(
		ctx,
		batchScopeKey{},
		scope,
	), batch
}
