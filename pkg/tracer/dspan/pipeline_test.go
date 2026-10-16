package dspan

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type recordOnlyTestSampler struct{}

func (recordOnlyTestSampler) ShouldSample(tracesdk.SamplingParameters) tracesdk.SamplingResult {
	return tracesdk.SamplingResult{
		Decision: tracesdk.RecordOnly,
	}
}

func (recordOnlyTestSampler) Description() string {
	return "RecordOnlyTest"
}

type errorPassProbe struct {
	tracesdk.SpanProcessor
}

type sampledProbeSpan struct {
	tracesdk.ReadOnlySpan
}

func (s sampledProbeSpan) SpanContext() trace.SpanContext {
	spanContext := s.ReadOnlySpan.SpanContext()

	return spanContext.WithTraceFlags(spanContext.TraceFlags() | trace.FlagsSampled)
}

func (p errorPassProbe) OnEnd(span tracesdk.ReadOnlySpan) {
	if span.SpanContext().IsSampled() {
		p.SpanProcessor.OnEnd(span)

		return
	}

	if span.Status().Code == codes.Error {
		p.SpanProcessor.OnEnd(
			sampledProbeSpan{
				ReadOnlySpan: span,
			},
		)
	}
}

type pipeline struct {
	exporter *tracetest.InMemoryExporter
	recorder *tracetest.SpanRecorder
}

func newPipeline(t *testing.T, sampled bool) *pipeline {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	recorder := tracetest.NewSpanRecorder()

	var sampler tracesdk.Sampler = recordOnlyTestSampler{}
	if sampled {
		sampler = tracesdk.AlwaysSample()
	}

	previous := otel.GetTracerProvider()
	provider := tracesdk.NewTracerProvider(
		tracesdk.WithSampler(sampler),
		tracesdk.WithSpanProcessor(recorder),
		tracesdk.WithSpanProcessor(
			errorPassProbe{
				SpanProcessor: tracesdk.NewSimpleSpanProcessor(exporter),
			},
		),
	)
	otel.SetTracerProvider(provider)

	t.Cleanup(
		func() {
			otel.SetTracerProvider(previous)
		},
	)

	return &pipeline{
		exporter: exporter,
		recorder: recorder,
	}
}

func (p *pipeline) exported() tracetest.SpanStubs {
	return p.exporter.GetSpans()
}

func (p *pipeline) exportedNames() []string {
	names := []string{}
	for _, span := range p.exporter.GetSpans() {
		names = append(
			names,
			span.Name,
		)
	}

	return names
}

func setFast(t *testing.T, enabled bool) {
	t.Helper()

	state := fastStateOff
	if enabled {
		state = fastStateOn
	}

	fastState.Store(state)
	t.Cleanup(
		func() {
			fastState.Store(fastStateUnset)
		},
	)
}

func loggedContext(t *testing.T, level log.Level) (context.Context, *bytes.Buffer) {
	t.Helper()

	buffer := &bytes.Buffer{}
	ctx := log.ContextWithLogger(
		t.Context(),
		log.New(
			log.WithWriter(buffer),
			log.WithLevel(level),
		),
	)

	return ctx, buffer
}

func logEntries(t *testing.T, buffer *bytes.Buffer) []map[string]any {
	t.Helper()

	entries := []map[string]any{}
	for _, line := range strings.Split(
		strings.TrimSpace(buffer.String()),
		"\n",
	) {
		if line == "" {
			continue
		}

		entry := map[string]any{}
		require.NoError(
			t,
			json.Unmarshal(
				[]byte(line),
				&entry,
			),
			line,
		)
		entries = append(
			entries,
			entry,
		)
	}

	return entries
}

func attributeMap(stub tracetest.SpanStub) map[string]string {
	attributes := map[string]string{}
	for _, attribute := range stub.Attributes {
		attributes[string(attribute.Key)] = attribute.Value.Emit()
	}

	return attributes
}

func sortedKeys(values map[string]any) []string {
	keys := make(
		[]string,
		0,
		len(values),
	)
	for key := range values {
		keys = append(
			keys,
			key,
		)
	}

	sort.Strings(keys)

	return keys
}
