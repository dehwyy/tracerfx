package nats

import (
	"context"

	"github.com/dehwyy/tracerfx/pkg/tracer/dspan"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const ConsumerSpanName = "nats.consumer"

// InjectTraceContext extracts trace context from ctx and injects it into NATS message headers.
func InjectTraceContext(ctx context.Context, msg *nats.Msg) {
	if msg.Header == nil {
		msg.Header = nats.Header{}
	}

	otel.GetTextMapPropagator().Inject(
		ctx,
		&NATSHeaderCarrier{Header: msg.Header},
	)
}

// ExtractTraceContext puts the remote span context from JetStream message headers into ctx.
// It does not start a span: use ConsumerSpan and end it after the handler returns.
// Compatible with brokerfx BeforeHandlerMiddleware signature.
func ExtractTraceContext(ctx context.Context, msg jetstream.Msg) (context.Context, error) {
	headers := msg.Headers()
	if headers == nil {
		return ctx, nil
	}

	return otel.GetTextMapPropagator().Extract(
		ctx,
		&NATSHeaderCarrier{Header: headers},
	), nil
}

// ExtractCore puts the remote span context from a core NATS message into ctx.
func ExtractCore(ctx context.Context, msg *nats.Msg) context.Context {
	if msg == nil || msg.Header == nil {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(
		ctx,
		&NATSHeaderCarrier{Header: msg.Header},
	)
}

// InjectMap writes trace context into m, e.g. outbox headers.
func InjectMap(ctx context.Context, m map[string]string) {
	if m == nil {
		return
	}

	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(m))
}

// ExtractMap reads trace context from m.
func ExtractMap(ctx context.Context, m map[string]string) context.Context {
	if len(m) == 0 {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(m))
}

// ConsumerSpan starts the consumer span; call the returned func after the handler finishes.
func ConsumerSpan(ctx context.Context, subject string) (context.Context, func()) {
	ctx, span := dspan.Start(
		ctx,
		ConsumerSpanName,
		dspan.Attr("subject", subject),
	)

	return ctx, span.End
}
