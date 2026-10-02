package middleware

import (
	"net/http"
	"strings"

	"github.com/dehwyy/tracerfx/pkg/tracer/dspan"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
)

const (
	TraceIDHeaderName = "X-Trace-Id"
	defaultSpanName   = "http.request"
)

type traceConfig struct {
	routeFunc    func(*http.Request) string
	spanNameFunc func(*http.Request, string) string
	propagate    bool
}

type TraceOption func(*traceConfig)

func WithRouteFunc(fn func(*http.Request) string) TraceOption {
	return func(c *traceConfig) {
		c.routeFunc = fn
	}
}

func WithPropagation(trust bool) TraceOption {
	return func(c *traceConfig) {
		c.propagate = trust
	}
}

func WithSpanNameFunc(fn func(r *http.Request, route string) string) TraceOption {
	return func(c *traceConfig) {
		c.spanNameFunc = fn
	}
}

func DefaultRoute(r *http.Request) string {
	pattern := r.Pattern
	if _, path, found := strings.Cut(pattern, " "); found {
		return path
	}
	return pattern
}

func defaultSpanNameFunc(r *http.Request, route string) string {
	if route == "" {
		return defaultSpanName
	}
	return "HTTP " + r.Method + " " + route
}

func TraceMiddleware(next http.Handler) http.Handler {
	return NewTrace()(next)
}

func NewTrace(opts ...TraceOption) func(http.Handler) http.Handler {
	cfg := traceConfig{
		routeFunc:    DefaultRoute,
		spanNameFunc: defaultSpanNameFunc,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				if cfg.propagate {
					ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
				}

				ctx, span := dspan.Start(
					ctx,
					defaultSpanName,
					dspan.Attr("method", r.Method),
					dspan.Attr("url", r.URL.String()),
				)
				defer span.End()

				w.Header().Set(TraceIDHeaderName, span.TraceID())

				recorder := &statusRecorder{ResponseWriter: w}
				handled := r.WithContext(ctx)
				next.ServeHTTP(recorder, handled)

				route := cfg.routeFunc(handled)
				status := recorder.statusCode()

				attrs := []attribute.KeyValue{
					attribute.String("http.request.method", r.Method),
					attribute.String("url.path", r.URL.Path),
					attribute.Int("http.response.status_code", status),
				}
				if route != "" {
					attrs = append(attrs, attribute.String("http.route", route))
				}
				span.TraceSpan.SetAttributes(attrs...)

				if name := cfg.spanNameFunc(handled, route); name != "" {
					span.SetName(name)
				}
				if status >= http.StatusInternalServerError {
					span.TraceSpan.SetStatus(codes.Error, http.StatusText(status))
				}
			},
		)
	}
}
