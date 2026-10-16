package grpc

import (
	"context"
	"os"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/dspan"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func init() {
	if err := os.Setenv(
		dspan.SpanDepthEnv,
		"batch",
	); err != nil {
		panic(err)
	}
}

func TestUnaryClientInterceptorInsideBatchCarriesBatchContext(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := tracesdk.NewTracerProvider(
		tracesdk.WithSpanProcessor(recorder),
	)
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(
		func() {
			otel.SetTracerProvider(previousProvider)
			otel.SetTextMapPropagator(previousPropagator)
		},
	)

	batchCtx, batch := dspan.StartBatch(
		t.Context(),
		"batch.process",
	)
	want := trace.SpanContextFromContext(batchCtx)
	require.True(
		t,
		want.IsValid(),
	)

	var outgoing metadata.MD
	invoker := func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		outgoing, _ = metadata.FromOutgoingContext(ctx)

		return nil
	}

	err := UnaryClientInterceptor()(
		batchCtx,
		"/payin.v1.Service/Create",
		nil,
		nil,
		nil,
		invoker,
	)
	require.NoError(
		t,
		err,
	)

	batch.End()

	values := outgoing.Get("traceparent")
	require.Len(
		t,
		values,
		1,
	)
	require.Contains(
		t,
		values[0],
		want.TraceID().String(),
	)
	require.Contains(
		t,
		values[0],
		want.SpanID().String(),
	)

	ended := recorder.Ended()
	require.Len(
		t,
		ended,
		1,
	)
	require.Equal(
		t,
		"batch.process",
		ended[0].Name(),
	)
}
