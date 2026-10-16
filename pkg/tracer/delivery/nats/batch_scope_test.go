package nats

import (
	"os"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/dspan"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	if err := os.Setenv(
		dspan.SpanDepthEnv,
		"batch",
	); err != nil {
		panic(err)
	}
}

func TestConsumerSpanInsideBatchPropagatesBatchContext(t *testing.T) {
	recorder, _, _ := setup(t)

	batchCtx, batch := dspan.StartBatch(
		t.Context(),
		"batch.process",
	)
	want := trace.SpanContextFromContext(batchCtx)
	require.True(
		t,
		want.IsValid(),
	)

	consumerCtx, done := ConsumerSpan(
		batchCtx,
		"payin.order.created",
	)

	headers := &nats.Msg{}
	InjectTraceContext(
		consumerCtx,
		headers,
	)

	carrier := map[string]string{}
	InjectMap(
		consumerCtx,
		carrier,
	)

	done()
	batch.End()

	require.NotEmpty(
		t,
		carrier["traceparent"],
	)

	for name, extracted := range map[string]trace.SpanContext{
		"map": trace.SpanContextFromContext(ExtractMap(
			t.Context(),
			carrier,
		)),
		"core": trace.SpanContextFromContext(ExtractCore(
			t.Context(),
			headers,
		)),
	} {
		require.True(
			t,
			extracted.IsValid(),
			name,
		)
		require.Equal(
			t,
			want.TraceID(),
			extracted.TraceID(),
			name,
		)
		require.Equal(
			t,
			want.SpanID(),
			extracted.SpanID(),
			name,
		)
		require.True(
			t,
			extracted.IsSampled(),
			name,
		)
	}

	names := make(
		[]string,
		0,
		len(recorder.Ended()),
	)
	for _, ended := range recorder.Ended() {
		names = append(
			names,
			ended.Name(),
		)
	}

	require.Contains(
		t,
		names,
		"batch.process",
	)
	require.NotContains(
		t,
		names,
		ConsumerSpanName,
	)
}
