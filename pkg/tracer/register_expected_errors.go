package tracer

import (
	"github.com/dehwyy/tracerfx/pkg/tracer/dspan"
)

func (o *options) registerExpectedErrors() {
	if o.expectedRegistered {
		return
	}

	o.expectedRegistered = true

	dspan.RegisterExpectedErrors(o.expectedErrors...)
}
