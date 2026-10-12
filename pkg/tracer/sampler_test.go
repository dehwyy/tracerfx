package tracer

import (
	"bytes"
	"context"
	"math"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestParseSampleRatio(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    float64
		wantErr bool
	}{
		{name: "empty means default", raw: "", want: 1},
		{name: "spaces mean default", raw: "  ", want: 1},
		{name: "ten percent", raw: "0.1", want: 0.1},
		{name: "padded", raw: " 0.25 ", want: 0.25},
		{name: "zero", raw: "0", want: 0},
		{name: "one", raw: "1", want: 1},
		{name: "negative", raw: "-0.1", want: 1, wantErr: true},
		{name: "above one", raw: "1.5", want: 1, wantErr: true},
		{name: "garbage", raw: "ten", want: 1, wantErr: true},
		{name: "nan", raw: "NaN", want: 1, wantErr: true},
		{name: "inf", raw: "Inf", want: 1, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ratio, err := parseSampleRatio(tc.raw)
			require.Equal(t, tc.want, ratio)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDefaultOptionsSampleRatioFromEnv(t *testing.T) {
	t.Setenv(SampleRatioEnv, "0.1")
	require.Equal(t, 0.1, newOptions().sampleRatio)
	require.NoError(t, newOptions().sampleRatioErr)

	t.Setenv(SampleRatioEnv, "bad")
	opts := newOptions()
	require.Equal(t, 1.0, opts.sampleRatio)
	require.Error(t, opts.sampleRatioErr)

	t.Setenv(SampleRatioEnv, "")
	require.Equal(t, 1.0, newOptions().sampleRatio)
}

func TestWithSampleRatioOverridesEnv(t *testing.T) {
	t.Setenv(SampleRatioEnv, "bad")

	opts := newOptions(WithSampleRatio(0.5))
	require.Equal(t, 0.5, opts.sampleRatio)
	require.NoError(t, opts.sampleRatioErr)

	opts = newOptions(WithSampleRatio(2))
	require.Equal(t, 1.0, opts.sampleRatio)
	require.Error(t, opts.sampleRatioErr)

	opts = newOptions(WithSampleRatio(math.NaN()))
	require.Equal(t, 1.0, opts.sampleRatio)
	require.Error(t, opts.sampleRatioErr)
}

func samplingParams(parent trace.SpanContext, traceID trace.TraceID) tracesdk.SamplingParameters {
	return tracesdk.SamplingParameters{
		ParentContext: trace.ContextWithRemoteSpanContext(context.Background(), parent),
		TraceID:       traceID,
		Name:          "span",
	}
}

func randomTraceID(seed uint64) trace.TraceID {
	var id trace.TraceID
	state := seed*0x9E3779B97F4A7C15 + 1
	for i := 0; i < 16; i += 8 {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		for j := 0; j < 8; j++ {
			id[i+j] = byte(state >> (8 * j))
		}
	}
	return id
}

func remoteParent(sampled bool) trace.SpanContext {
	flags := trace.TraceFlags(0)
	if sampled {
		flags = trace.FlagsSampled
	}
	return trace.NewSpanContext(
		trace.SpanContextConfig{
			TraceID:    randomTraceID(1),
			SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
			TraceFlags: flags,
			Remote:     true,
		},
	)
}

func TestSamplerRatioOneIsAlwaysSample(t *testing.T) {
	sampler := newSampler(1)

	require.Equal(t, tracesdk.AlwaysSample().Description(), sampler.Description())

	for seed := uint64(0); seed < 200; seed++ {
		require.Equal(
			t,
			tracesdk.RecordAndSample,
			sampler.ShouldSample(samplingParams(trace.SpanContext{}, randomTraceID(seed))).Decision,
		)
	}

	require.Equal(
		t,
		tracesdk.RecordAndSample,
		sampler.ShouldSample(samplingParams(remoteParent(false), randomTraceID(7))).Decision,
	)
}

func TestSamplerRatioZeroDropsRoots(t *testing.T) {
	sampler := newSampler(0)

	for seed := uint64(0); seed < 200; seed++ {
		require.Equal(
			t,
			tracesdk.Drop,
			sampler.ShouldSample(samplingParams(trace.SpanContext{}, randomTraceID(seed))).Decision,
		)
	}
}

func TestSamplerRatioFollowsParent(t *testing.T) {
	sampler := newSampler(0.1)

	for seed := uint64(0); seed < 200; seed++ {
		require.Equal(
			t,
			tracesdk.RecordAndSample,
			sampler.ShouldSample(samplingParams(remoteParent(true), randomTraceID(seed))).Decision,
		)
		require.Equal(
			t,
			tracesdk.Drop,
			sampler.ShouldSample(samplingParams(remoteParent(false), randomTraceID(seed))).Decision,
		)
	}
}

func TestSamplerRatioApproximatesFraction(t *testing.T) {
	sampler := newSampler(0.1)

	const total = 20000
	sampled := 0
	for seed := uint64(0); seed < total; seed++ {
		if sampler.ShouldSample(samplingParams(trace.SpanContext{}, randomTraceID(seed))).Decision == tracesdk.RecordAndSample {
			sampled++
		}
	}

	fraction := float64(sampled) / total
	require.InDelta(t, 0.1, fraction, 0.02)
}

func TestProviderStartUsesSampleRatio(t *testing.T) {
	t.Setenv(SampleRatioEnv, "0")

	provider := NewProvider(WithHost("127.0.0.1:1"))
	require.NoError(t, provider.Start(t.Context()))
	t.Cleanup(func() { require.NoError(t, provider.Stop(context.Background())) })

	for i := 0; i < 50; i++ {
		_, span := otel.Tracer("test").Start(context.Background(), "root")
		require.False(t, span.SpanContext().IsSampled())
		span.End()
	}
}

func TestProviderStartDefaultSamplesEverything(t *testing.T) {
	t.Setenv(SampleRatioEnv, "")

	provider := NewProvider(WithHost("127.0.0.1:1"))
	require.NoError(t, provider.Start(t.Context()))
	t.Cleanup(func() { _ = provider.provider.Shutdown(context.Background()) })

	_, span := otel.Tracer("test").Start(context.Background(), "root")
	require.True(t, span.SpanContext().IsSampled())
}

func TestProviderStartWarnsOnInvalidRatio(t *testing.T) {
	t.Setenv(SampleRatioEnv, "bad")

	var buf bytes.Buffer
	provider := NewProvider(
		WithHost("127.0.0.1:1"),
		WithLogger(log.New(log.WithWriter(&buf))),
	)
	require.NoError(t, provider.Start(t.Context()))
	t.Cleanup(func() { _ = provider.provider.Shutdown(context.Background()) })

	require.Contains(t, buf.String(), "tracer sample ratio ignored")
	require.Contains(t, buf.String(), SampleRatioEnv)
}
