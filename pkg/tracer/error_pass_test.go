package tracer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type probeProcessor struct {
	started  int
	ended    []tracesdk.ReadOnlySpan
	flushErr error
	stopErr  error
	flushed  int
	stopped  int
}

func (p *probeProcessor) OnStart(_ context.Context, _ tracesdk.ReadWriteSpan) {
	p.started++
}

func (p *probeProcessor) OnEnd(span tracesdk.ReadOnlySpan) {
	p.ended = append(
		p.ended,
		span,
	)
}

func (p *probeProcessor) Shutdown(_ context.Context) error {
	p.stopped++

	return p.stopErr
}

func (p *probeProcessor) ForceFlush(_ context.Context) error {
	p.flushed++

	return p.flushErr
}

func stubSpan(name string, sampled bool, code codes.Code) tracesdk.ReadOnlySpan {
	flags := trace.TraceFlags(0)
	if sampled {
		flags = trace.FlagsSampled
	}

	stub := tracetest.SpanStub{
		Name: name,
		SpanContext: trace.NewSpanContext(
			trace.SpanContextConfig{
				TraceID:    randomTraceID(3),
				SpanID:     trace.SpanID{9, 8, 7, 6, 5, 4, 3, 2},
				TraceFlags: flags,
			},
		),
		Status: tracesdk.Status{
			Code: code,
		},
	}

	return stub.Snapshot()
}

func TestErrorPassProcessorOnEnd(t *testing.T) {
	cases := []struct {
		name        string
		sampled     bool
		code        codes.Code
		wantForward bool
		wantSame    bool
	}{
		{
			name:        "sampled ok",
			sampled:     true,
			code:        codes.Ok,
			wantForward: true,
			wantSame:    true,
		},
		{
			name:        "sampled unset",
			sampled:     true,
			code:        codes.Unset,
			wantForward: true,
			wantSame:    true,
		},
		{
			name:        "sampled error",
			sampled:     true,
			code:        codes.Error,
			wantForward: true,
			wantSame:    true,
		},
		{
			name:    "unsampled ok",
			sampled: false,
			code:    codes.Ok,
		},
		{
			name:    "unsampled unset",
			sampled: false,
			code:    codes.Unset,
		},
		{
			name:        "unsampled error",
			sampled:     false,
			code:        codes.Error,
			wantForward: true,
		},
	}

	for _, tc := range cases {
		t.Run(
			tc.name,
			func(t *testing.T) {
				probe := &probeProcessor{}
				processor := newErrorPassProcessor(probe)
				span := stubSpan(
					"op",
					tc.sampled,
					tc.code,
				)

				processor.OnEnd(span)

				if !tc.wantForward {
					require.Empty(
						t,
						probe.ended,
					)
					return
				}

				require.Len(
					t,
					probe.ended,
					1,
				)
				forwarded := probe.ended[0]
				require.True(
					t,
					forwarded.SpanContext().IsSampled(),
				)
				require.Equal(
					t,
					span.Name(),
					forwarded.Name(),
				)
				require.Equal(
					t,
					span.SpanContext().TraceID(),
					forwarded.SpanContext().TraceID(),
				)
				require.Equal(
					t,
					span.SpanContext().SpanID(),
					forwarded.SpanContext().SpanID(),
				)
				require.Equal(
					t,
					tc.code,
					forwarded.Status().Code,
				)

				if tc.wantSame {
					require.Equal(
						t,
						span,
						forwarded,
					)
				}
			},
		)
	}
}

func TestErrorPassProcessorKeepsOriginalSpanUnsampled(t *testing.T) {
	probe := &probeProcessor{}
	span := stubSpan(
		"op",
		false,
		codes.Error,
	)

	newErrorPassProcessor(probe).OnEnd(span)

	require.False(
		t,
		span.SpanContext().IsSampled(),
	)
}

func TestErrorPassProcessorPassesThroughLifecycle(t *testing.T) {
	flushErr := errors.New("flush failed")
	stopErr := errors.New("stop failed")
	probe := &probeProcessor{
		flushErr: flushErr,
		stopErr:  stopErr,
	}
	processor := newErrorPassProcessor(probe)

	processor.OnStart(
		t.Context(),
		nil,
	)
	require.Equal(
		t,
		1,
		probe.started,
	)

	require.ErrorIs(
		t,
		processor.ForceFlush(t.Context()),
		flushErr,
	)
	require.Equal(
		t,
		1,
		probe.flushed,
	)

	require.ErrorIs(
		t,
		processor.Shutdown(t.Context()),
		stopErr,
	)
	require.Equal(
		t,
		1,
		probe.stopped,
	)
}

func newExportingProvider(t *testing.T, ratio float64) (*tracesdk.TracerProvider, *tracetest.InMemoryExporter) {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	provider := tracesdk.NewTracerProvider(
		tracesdk.WithSpanProcessor(
			newBatcher(
				exporter,
				ratio,
			),
		),
		tracesdk.WithSampler(newSampler(ratio)),
	)
	t.Cleanup(
		func() {
			require.NoError(
				t,
				provider.Shutdown(context.Background()),
			)
		},
	)

	return provider, exporter
}

func exportedNames(t *testing.T, provider *tracesdk.TracerProvider, exporter *tracetest.InMemoryExporter) []string {
	t.Helper()

	require.NoError(
		t,
		provider.ForceFlush(context.Background()),
	)

	names := []string{}
	for _, span := range exporter.GetSpans() {
		names = append(
			names,
			span.Name,
		)
	}

	return names
}

func TestRatioZeroSpansRecordWithoutSampling(t *testing.T) {
	provider, _ := newExportingProvider(
		t,
		0,
	)
	tracer := provider.Tracer("test")

	ctx, root := tracer.Start(
		context.Background(),
		"root",
	)
	_, child := tracer.Start(
		ctx,
		"child",
	)

	require.True(
		t,
		root.IsRecording(),
	)
	require.False(
		t,
		root.SpanContext().IsSampled(),
	)
	require.True(
		t,
		child.IsRecording(),
	)
	require.False(
		t,
		child.SpanContext().IsSampled(),
	)
	require.Equal(
		t,
		root.SpanContext().TraceID(),
		child.SpanContext().TraceID(),
	)

	child.End()
	root.End()
}

func TestRatioZeroExportsOnlyErrorSpans(t *testing.T) {
	provider, exporter := newExportingProvider(
		t,
		0,
	)
	tracer := provider.Tracer("test")

	ctx, root := tracer.Start(
		context.Background(),
		"root",
	)
	_, failed := tracer.Start(
		ctx,
		"failed",
	)
	failed.SetStatus(
		codes.Error,
		"boom",
	)
	failed.End()
	_, fine := tracer.Start(
		ctx,
		"fine",
	)
	fine.SetStatus(
		codes.Ok,
		"",
	)
	fine.End()
	_, plain := tracer.Start(
		ctx,
		"plain",
	)
	plain.End()
	root.End()

	require.Equal(
		t,
		[]string{"failed"},
		exportedNames(
			t,
			provider,
			exporter,
		),
	)

	exported := exporter.GetSpans()[0]
	require.True(
		t,
		exported.SpanContext.IsSampled(),
	)
	require.Equal(
		t,
		codes.Error,
		exported.Status.Code,
	)
	require.Equal(
		t,
		"boom",
		exported.Status.Description,
	)
}

func TestRatioZeroRemoteParents(t *testing.T) {
	provider, exporter := newExportingProvider(
		t,
		0,
	)
	tracer := provider.Tracer("test")

	unsampled := trace.ContextWithRemoteSpanContext(
		context.Background(),
		remoteParent(false),
	)
	_, quiet := tracer.Start(
		unsampled,
		"quiet",
	)
	require.True(
		t,
		quiet.IsRecording(),
	)
	require.False(
		t,
		quiet.SpanContext().IsSampled(),
	)
	quiet.End()

	_, loud := tracer.Start(
		unsampled,
		"loud",
	)
	loud.SetStatus(
		codes.Error,
		"boom",
	)
	loud.End()

	sampled := trace.ContextWithRemoteSpanContext(
		context.Background(),
		remoteParent(true),
	)
	_, followed := tracer.Start(
		sampled,
		"followed",
	)
	require.True(
		t,
		followed.IsRecording(),
	)
	require.True(
		t,
		followed.SpanContext().IsSampled(),
	)
	followed.End()

	require.ElementsMatch(
		t,
		[]string{"loud", "followed"},
		exportedNames(
			t,
			provider,
			exporter,
		),
	)
}

func TestRatioOneExportsEverything(t *testing.T) {
	provider, exporter := newExportingProvider(
		t,
		1,
	)
	tracer := provider.Tracer("test")

	ctx, root := tracer.Start(
		context.Background(),
		"root",
	)
	_, failed := tracer.Start(
		ctx,
		"failed",
	)
	failed.SetStatus(
		codes.Error,
		"boom",
	)
	failed.End()
	_, fine := tracer.Start(
		ctx,
		"fine",
	)
	fine.End()
	root.End()

	require.True(
		t,
		root.SpanContext().IsSampled(),
	)
	require.ElementsMatch(
		t,
		[]string{"root", "failed", "fine"},
		exportedNames(
			t,
			provider,
			exporter,
		),
	)
}

func TestPartialRatioExportsSampledTracesOnceAndAllErrors(t *testing.T) {
	provider, exporter := newExportingProvider(
		t,
		0.1,
	)
	tracer := provider.Tracer("test")

	const traces = 4000
	const errorEvery = 10

	for i := 0; i < traces; i++ {
		ctx, root := tracer.Start(
			context.Background(),
			"root",
		)
		_, child := tracer.Start(
			ctx,
			"child",
		)
		if i%errorEvery == 0 {
			child.SetStatus(
				codes.Error,
				"boom",
			)
		}
		child.End()
		root.End()
	}

	require.NoError(
		t,
		provider.ForceFlush(context.Background()),
	)

	roots := 0
	failedChildren := 0
	for _, span := range exporter.GetSpans() {
		switch span.Name {
		case "root":
			roots++
		case "child":
			if span.Status.Code == codes.Error {
				failedChildren++
			}
		}
	}

	require.InDelta(
		t,
		0.1,
		float64(roots)/traces,
		0.03,
	)
	require.Equal(
		t,
		traces/errorEvery,
		failedChildren,
	)

	seen := map[string]int{}
	for _, span := range exporter.GetSpans() {
		seen[span.SpanContext.SpanID().String()]++
	}
	for spanID, count := range seen {
		require.Equal(
			t,
			1,
			count,
			spanID,
		)
	}
}

func TestSamplerDescriptionMentionsRatio(t *testing.T) {
	require.Contains(
		t,
		newSampler(0.1).Description(),
		"TraceIDRatioBased{0.1}",
	)
	require.Contains(
		t,
		newSampler(0.25).Description(),
		"0.25",
	)
	require.Contains(
		t,
		newSampler(0.1).Description(),
		"RecordOnlyUnsampled",
	)
}
