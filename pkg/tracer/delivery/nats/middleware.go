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

func InjectTraceContext(ctx context.Context, msg *nats.Msg) {
	if msg.Header == nil {
		msg.Header = nats.Header{}
	}

	otel.GetTextMapPropagator().Inject(
		ctx,
		&NATSHeaderCarrier{Header: msg.Header},
	)
}

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

func ExtractCore(ctx context.Context, msg *nats.Msg) context.Context {
	if msg == nil || msg.Header == nil {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(
		ctx,
		&NATSHeaderCarrier{Header: msg.Header},
	)
}

func InjectMap(ctx context.Context, m map[string]string) {
	if m == nil {
		return
	}

	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(m))
}

func ExtractMap(ctx context.Context, m map[string]string) context.Context {
	if len(m) == 0 {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(m))
}

func ConsumerSpan(ctx context.Context, subject string) (context.Context, func()) {
	ctx, span := dspan.Start(
		ctx,
		ConsumerSpanName,
		dspan.Attr("subject", subject),
	)

	return ctx, span.End
}
