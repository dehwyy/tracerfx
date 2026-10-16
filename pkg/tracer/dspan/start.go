package dspan

import (
	"context"
)

func Start(ctx context.Context, spanName string, attrs ...Attribute) (context.Context, *dspan) {
	return begin(
		ctx,
		spanName,
		attrs,
		callerSkipStart,
	)
}
