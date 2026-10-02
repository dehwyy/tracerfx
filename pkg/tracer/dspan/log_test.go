package dspan

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

func TestSpanLogsCarryTraceIDOnceAndErrorText(t *testing.T) {
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(tracesdk.NewTracerProvider())
	t.Cleanup(func() { otel.SetTracerProvider(previous) })

	tests := []struct {
		name string
		run  func(span *dspan)
		want []string
	}{
		{
			name: "error",
			run:  func(span *dspan) { span.Err(errors.New("boom")) },
			want: []string{`"error":"boom"`, `"span_name":"op"`},
		},
		{
			name: "end",
			run:  func(span *dspan) { span.End() },
			want: []string{`"message":"Span ended"`, `"duration_ms"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := log.ContextWithLogger(t.Context(), log.New(log.WithWriter(&buf), log.WithLevel(log.LevelDebug)))

			_, span := Start(ctx, "op")
			buf.Reset()
			tt.run(span)

			out := buf.String()
			require.Equal(t, 1, strings.Count(out, `"trace_id"`), out)
			require.Equal(t, 1, strings.Count(out, `"span_id"`), out)
			for _, want := range tt.want {
				require.Contains(t, out, want)
			}
		})
	}
}

func TestSetName(t *testing.T) {
	var buf bytes.Buffer
	ctx := log.ContextWithLogger(t.Context(), log.New(log.WithWriter(&buf), log.WithLevel(log.LevelDebug)))

	_, span := Start(ctx, "old")
	span.SetName("new")
	buf.Reset()
	span.End()

	require.Contains(t, buf.String(), `"span_name":"new"`)
}
