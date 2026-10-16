package dspan

import (
	"go.opentelemetry.io/otel/codes"
)

func (c *batchScope) recordFailure(err error) {
	first := c.failed.CompareAndSwap(
		false,
		true,
	)

	c.batch.TraceSpan.RecordError(err)

	if !first {
		return
	}

	c.batch.Mu.Lock()
	c.batch.expandPending()
	c.batch.Mu.Unlock()

	c.batch.TraceSpan.SetStatus(
		codes.Error,
		err.Error(),
	)
}
