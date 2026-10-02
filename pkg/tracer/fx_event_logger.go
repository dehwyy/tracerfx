package tracer

import (
	"github.com/dehwyy/tracerfx/pkg/tracer/log"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func FxEventLogger() fx.Option {
	return fx.WithLogger(
		func(l log.Logger) fxevent.Logger {
			return log.NewFxEventLogger(l)
		},
	)
}
