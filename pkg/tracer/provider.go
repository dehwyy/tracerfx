package tracer

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Provider struct {
	options  *options
	exporter *otlptrace.Exporter
	provider *tracesdk.TracerProvider
}

func NewProvider(opts ...Option) *Provider {
	return &Provider{
		options: newOptions(opts...),
	}
}

func (p *Provider) Start(ctx context.Context) error {
	var err error

	p.exporter, err = otlptracegrpc.New(
		ctx,
		append(
			[]otlptracegrpc.Option{
				otlptracegrpc.WithEndpoint(p.options.host),
				otlptracegrpc.WithInsecure(),
			}, p.options.traceGrpcOptions...,
		)...,
	)
	if err != nil {
		return err
	}

	res, err := resource.New(
		ctx,
		append(
			[]resource.Option{
				resource.WithAttributes(
					semconv.ServiceNameKey.String(p.options.serviceName),
					semconv.ServiceVersionKey.String(p.options.serviceVersion),
				),
			},
			p.options.resourceOptions...,
		)...,
	)
	if err != nil {
		return err
	}

	p.provider = tracesdk.NewTracerProvider(
		append(
			[]tracesdk.TracerProviderOption{
				tracesdk.WithBatcher(p.exporter),
				tracesdk.WithResource(res),
				tracesdk.WithSampler(tracesdk.AlwaysSample()),
			}, p.options.tracerProviderOptions...,
		)...,
	)

	otel.SetTracerProvider(p.provider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return nil
}

func (p *Provider) Stop(ctx context.Context) error {
	if p.provider != nil {
		if err := p.provider.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}
