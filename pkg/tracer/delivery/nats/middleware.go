package nats

import (
	"context"

	"github.com/dehwyy/tracerfx/pkg/tracer/dspan"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
)

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

// ExtractTraceContext extracts trace context from NATS JetStream message headers into context.
// Compatible with brokerfx BeforeHandlerMiddleware signature.
func ExtractTraceContext(ctx context.Context, msg jetstream.Msg) (context.Context, error) {
	headers := msg.Headers()
	if headers == nil {
		return ctx, nil
	}

	ctx = otel.GetTextMapPropagator().Extract(
		ctx,
		&NATSHeaderCarrier{Header: headers},
	)

	ctx, span := dspan.Start(
		ctx,
		"nats.consumer",
		dspan.Attr("subject", msg.Subject()),
	)
	defer span.End()

	return ctx, nil
}
