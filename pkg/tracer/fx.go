package tracer

import (
	"context"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"

	"go.uber.org/fx"
)

// FxModule provides the tracer Provider and Logger to the Uber FX dependency injection container.
func FxModule(tracerOpts ...Option) fx.Option {
	log.SetGlobalLevelFromEnv()

	opts := newOptions(tracerOpts...)
	log.SetDefaultLogger(opts.logger)
	opts.registerExpectedErrors()

	return fx.Module("tracer",
		fx.Provide(
			func() log.Logger {
				return opts.logger
			},
			func(lc fx.Lifecycle) *Provider {
				provider := newProvider(opts)

				lc.Append(fx.Hook{
					OnStart: func(ctx context.Context) error {
						return provider.Start(ctx)
					},
					OnStop: func(ctx context.Context) error {
						return provider.Stop(ctx)
					},
				})

				return provider
			},
		),
		fx.Invoke(func(p *Provider) {}),
	)
}
