package tracer

import (
	"github.com/dehwyy/tracerfx/pkg/tracer/log"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

type options struct {
	host                  string
	serviceName           string
	serviceVersion        string
	logger                log.Logger
	traceGrpcOptions      []otlptracegrpc.Option
	resourceOptions       []resource.Option
	tracerProviderOptions []tracesdk.TracerProviderOption
}

func newOptions(opts ...Option) *options {
	o := defaultOptions()

	for _, opt := range opts {
		opt.apply(&o)
	}

	return &o
}

// defaultOptions by default we use basic values, you can extend this to use git.GetCommitInfo if available.
func defaultOptions() options {
	return options{
		host:           "localhost:4317",
		serviceName:    "unknown-service",
		serviceVersion: "v0.0.0",
		logger:         log.NewZerologLogger(),
	}
}

// Option overrides behavior of Provider.
type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

func WithHost(url string) Option {
	return optionFunc(
		func(o *options) {
			o.host = url
		},
	)
}

func WithServiceName(serviceName string) Option {
	return optionFunc(
		func(o *options) {
			o.serviceName = serviceName
		},
	)
}

func WithServiceVersion(serviceVersion string) Option {
	return optionFunc(
		func(o *options) {
			o.serviceVersion = serviceVersion
		},
	)
}

func WithLogger(logger log.Logger) Option {
	return optionFunc(
		func(o *options) {
			o.logger = logger
		},
	)
}

func WithOTLPTraceGrpcOptions(opts ...otlptracegrpc.Option) Option {
	return optionFunc(
		func(o *options) {
			o.traceGrpcOptions = append(o.traceGrpcOptions, opts...)
		},
	)
}

func WithResourceOptions(opts ...resource.Option) Option {
	return optionFunc(
		func(o *options) {
			o.resourceOptions = append(o.resourceOptions, opts...)
		},
	)
}

func WithTracerProviderOptions(opts ...tracesdk.TracerProviderOption) Option {
	return optionFunc(
		func(o *options) {
			o.tracerProviderOptions = append(o.tracerProviderOptions, opts...)
		},
	)
}
