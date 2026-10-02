package nats

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type fakeJetstreamMsg struct {
	jetstream.Msg
	headers nats.Header
	subject string
}

func (f fakeJetstreamMsg) Headers() nats.Header { return f.headers }
func (f fakeJetstreamMsg) Subject() string      { return f.subject }

func setup(t *testing.T) (*tracetest.SpanRecorder, context.Context, trace.SpanContext) {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	provider := tracesdk.NewTracerProvider(tracesdk.WithSpanProcessor(recorder))
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	ctx, span := provider.Tracer("test").Start(t.Context(), "producer")
	t.Cleanup(func() { span.End() })

	return recorder, ctx, span.SpanContext()
}

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		run  func(ctx context.Context) context.Context
	}{
		{
			name: "core msg",
			run: func(ctx context.Context) context.Context {
				msg := &nats.Msg{}
				InjectTraceContext(ctx, msg)
				return ExtractCore(context.Background(), msg)
			},
		},
		{
			name: "jetstream msg",
			run: func(ctx context.Context) context.Context {
				msg := &nats.Msg{}
				InjectTraceContext(ctx, msg)
				got, err := ExtractTraceContext(context.Background(), fakeJetstreamMsg{headers: msg.Header, subject: "s"})
				require.NoError(t, err)
				return got
			},
		},
		{
			name: "map",
			run: func(ctx context.Context) context.Context {
				carrier := map[string]string{}
				InjectMap(ctx, carrier)
				require.NotEmpty(t, carrier["traceparent"])
				return ExtractMap(context.Background(), carrier)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ctx, want := setup(t)

			got := trace.SpanContextFromContext(tt.run(ctx))

			require.True(t, got.IsValid())
			require.True(t, got.IsRemote())
			require.Equal(t, want.TraceID(), got.TraceID())
			require.Equal(t, want.SpanID(), got.SpanID())
		})
	}
}

func TestExtractWithoutCarrier(t *testing.T) {
	tests := []struct {
		name string
		run  func(ctx context.Context) context.Context
	}{
		{name: "core nil msg", run: func(ctx context.Context) context.Context { return ExtractCore(ctx, nil) }},
		{name: "core nil header", run: func(ctx context.Context) context.Context { return ExtractCore(ctx, &nats.Msg{}) }},
		{name: "map nil", run: func(ctx context.Context) context.Context { return ExtractMap(ctx, nil) }},
		{name: "map empty", run: func(ctx context.Context) context.Context { return ExtractMap(ctx, map[string]string{}) }},
		{
			name: "jetstream nil headers",
			run: func(ctx context.Context) context.Context {
				got, err := ExtractTraceContext(ctx, fakeJetstreamMsg{})
				require.NoError(t, err)
				return got
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ctx, _ := setup(t)
			require.Equal(t, ctx, tt.run(ctx))
		})
	}
}

func TestInjectMapNilIsNoop(t *testing.T) {
	_, ctx, _ := setup(t)
	require.NotPanics(t, func() { InjectMap(ctx, nil) })
}

func TestExtractDoesNotStartOrEndSpan(t *testing.T) {
	recorder, ctx, _ := setup(t)

	msg := &nats.Msg{}
	InjectTraceContext(ctx, msg)
	_, err := ExtractTraceContext(context.Background(), fakeJetstreamMsg{headers: msg.Header, subject: "s"})

	require.NoError(t, err)
	require.Empty(t, recorder.Ended())
	require.Empty(t, recorder.Started()[1:])
}

func TestConsumerSpanLivesUntilDoneIsCalled(t *testing.T) {
	recorder, producerCtx, producer := setup(t)

	msg := &nats.Msg{}
	InjectTraceContext(producerCtx, msg)
	ctx, err := ExtractTraceContext(context.Background(), fakeJetstreamMsg{headers: msg.Header, subject: "orders.created"})
	require.NoError(t, err)

	ctx, done := ConsumerSpan(ctx, "orders.created")
	_, child := otel.Tracer("test").Start(ctx, "handler")
	child.End()

	require.Len(t, recorder.Ended(), 1)
	require.Equal(t, "handler", recorder.Ended()[0].Name())

	done()

	ended := recorder.Ended()
	require.Len(t, ended, 2)
	consumer := ended[1]
	require.Equal(t, ConsumerSpanName, consumer.Name())
	require.Equal(t, producer.TraceID(), consumer.SpanContext().TraceID())
	require.Equal(t, producer.SpanID(), consumer.Parent().SpanID())
	require.Equal(t, consumer.SpanContext().SpanID(), ended[0].Parent().SpanID())
}
