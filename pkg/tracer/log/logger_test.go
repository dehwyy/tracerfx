package log

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

type stringerValue struct{}

func (stringerValue) String() string { return "stringer-value" }

type nilStringer struct{}

func (*nilStringer) String() string { return "never" }

func decode(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()

	var entries []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		entry := map[string]any{}
		require.NoError(t, json.Unmarshal([]byte(line), &entry), line)
		delete(entry, "time")
		entries = append(entries, entry)
	}
	return entries
}

func TestLevels(t *testing.T) {
	tests := []struct {
		name      string
		level     Level
		wantLines []string
	}{
		{name: "debug", level: LevelDebug, wantLines: []string{"debug", "info", "warn", "error"}},
		{name: "info", level: LevelInfo, wantLines: []string{"info", "warn", "error"}},
		{name: "warn", level: LevelWarn, wantLines: []string{"warn", "error"}},
		{name: "error", level: LevelError, wantLines: []string{"error"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := New(WithWriter(&buf), WithLevel(tt.level))

			l.Debug("m")
			l.Info("m")
			l.Warn("m")
			l.Error("m")

			var got []string
			for _, entry := range decode(t, &buf) {
				got = append(got, entry["level"].(string))
			}
			require.Equal(t, tt.wantLines, got)
		})
	}
}

func TestFields(t *testing.T) {
	when := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	var nilPtr *nilStringer

	tests := []struct {
		name string
		args []any
		want map[string]any
	}{
		{
			name: "error is written as text",
			args: []any{"error", errors.New("boom")},
			want: map[string]any{"error": "boom"},
		},
		{
			name: "wrapped error",
			args: []any{"error", fmt.Errorf("ctx: %w", errors.New("boom"))},
			want: map[string]any{"error": "ctx: boom"},
		},
		{
			name: "Err attr",
			args: []any{Err(errors.New("boom"))},
			want: map[string]any{"error": "boom"},
		},
		{
			name: "time",
			args: []any{"at", when},
			want: map[string]any{"at": "2026-10-02T12:00:00Z"},
		},
		{
			name: "duration",
			args: []any{"took", 1500 * time.Millisecond},
			want: map[string]any{"took": float64(1500)},
		},
		{
			name: "stringer",
			args: []any{"s", stringerValue{}},
			want: map[string]any{"s": "stringer-value"},
		},
		{
			name: "nil stringer pointer",
			args: []any{"s", nilPtr},
			want: map[string]any{"s": nil},
		},
		{
			name: "bytes",
			args: []any{"b", []byte("raw")},
			want: map[string]any{"b": "raw"},
		},
		{
			name: "scalars",
			args: []any{"i", 7, "b", true, "f", 1.5, "s", "x"},
			want: map[string]any{"i": float64(7), "b": true, "f": 1.5, "s": "x"},
		},
		{
			name: "odd tail",
			args: []any{"a", 1, "dangling"},
			want: map[string]any{"a": float64(1), "!BADKEY": "dangling"},
		},
		{
			name: "non string key",
			args: []any{42, "a", "b"},
			want: map[string]any{"!BADKEY": float64(42), "a": "b"},
		},
		{
			name: "struct",
			args: []any{"obj", struct{ A int }{A: 1}},
			want: map[string]any{"obj": map[string]any{"A": float64(1)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := New(WithWriter(&buf), WithLevel(LevelDebug))

			l.Info("msg", tt.args...)
			l.With(tt.args...).Info("msg")

			entries := decode(t, &buf)
			require.Len(t, entries, 2)
			for _, entry := range entries {
				require.Equal(t, "msg", entry["message"])
				require.Equal(t, "info", entry["level"])
				for key, want := range tt.want {
					require.Equal(t, want, entry[key], key)
				}
			}
		})
	}
}

func TestWithKeepsLevel(t *testing.T) {
	var buf bytes.Buffer
	l := New(WithWriter(&buf), WithLevel(LevelInfo)).With("k", "v")

	l.Debug("hidden")
	l.Info("shown")

	entries := decode(t, &buf)
	require.Len(t, entries, 1)
	require.Equal(t, "v", entries[0]["k"])
}

func TestDeprecatedDebugHelper(t *testing.T) {
	tests := []struct {
		name  string
		level Level
		want  int
	}{
		{name: "dropped at info", level: LevelInfo, want: 0},
		{name: "written at debug", level: LevelDebug, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Debug(New(WithWriter(&buf), WithLevel(tt.level)), "Span started", "span_name", "x")
			require.Len(t, decode(t, &buf), tt.want)
		})
	}
}

func spanContext(t *testing.T, valid bool) trace.SpanContext {
	t.Helper()
	if !valid {
		return trace.SpanContext{}
	}
	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	require.NoError(t, err)
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
}

func TestFromContextTraceCorrelation(t *testing.T) {
	tests := []struct {
		name      string
		validSpan bool
		wantIDs   bool
	}{
		{name: "valid span adds ids", validSpan: true, wantIDs: true},
		{name: "no span no ids", validSpan: false, wantIDs: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := ContextWithLogger(t.Context(), New(WithWriter(&buf)))
			ctx = trace.ContextWithSpanContext(ctx, spanContext(t, tt.validSpan))

			FromContext(ctx).Info("msg")
			FromContext(ctx).With("k", "v").Info("msg")

			entries := decode(t, &buf)
			require.Len(t, entries, 2)
			for _, entry := range entries {
				if !tt.wantIDs {
					require.NotContains(t, entry, "trace_id")
					require.NotContains(t, entry, "span_id")
					continue
				}
				require.Equal(t, "0102030405060708090a0b0c0d0e0f10", entry["trace_id"])
				require.Equal(t, "0102030405060708", entry["span_id"])
			}
		})
	}
}

func TestFromContextDoesNotDuplicateIDs(t *testing.T) {
	var buf bytes.Buffer
	ctx := trace.ContextWithSpanContext(t.Context(), spanContext(t, true))
	ctx = ContextWithLogger(ctx, New(WithWriter(&buf)))
	ctx = ContextWithLogger(ctx, FromContext(ctx))

	FromContext(ctx).Info("msg")

	require.Equal(t, 1, strings.Count(buf.String(), `"trace_id"`))
}

func TestFromContextNestedSpansNoDuplicateKeys(t *testing.T) {
	var buf bytes.Buffer
	parentID, err := trace.SpanIDFromHex("0102030405060708")
	require.NoError(t, err)
	childID, err := trace.SpanIDFromHex("1112131415161718")
	require.NoError(t, err)
	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	require.NoError(t, err)

	parent := trace.ContextWithSpanContext(t.Context(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     parentID,
		TraceFlags: trace.FlagsSampled,
	}))
	parent = ContextWithLogger(parent, New(WithWriter(&buf)))
	parent = ContextWithLogger(parent, FromContext(parent).With("k", "v"))

	child := trace.ContextWithSpanContext(parent, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     childID,
		TraceFlags: trace.FlagsSampled,
	}))

	FromContext(child).Info("msg")
	FromContext(child).With("a", "b").Info("msg")

	require.Equal(t, 2, strings.Count(buf.String(), `"trace_id"`))
	require.Equal(t, 2, strings.Count(buf.String(), `"span_id"`))

	for _, entry := range decode(t, &buf) {
		require.Equal(t, childID.String(), entry["span_id"])
		require.Equal(t, "v", entry["k"])
	}
}

func TestDefault(t *testing.T) {
	t.Cleanup(func() { SetDefaultLogger(nil) })

	var buf bytes.Buffer
	SetDefaultLogger(New(WithWriter(&buf)))

	Default().Warn("from default")
	FromContext(t.Context()).Warn("from context")
	FromContext(nil).Warn("from nil context")

	require.Len(t, decode(t, &buf), 3)
}

func TestDefaultFallbackIsStable(t *testing.T) {
	SetDefaultLogger(nil)
	require.NotNil(t, Default())
	require.Same(t, Default(), Default())
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in      string
		want    Level
		wantErr bool
	}{
		{in: "debug", want: LevelDebug},
		{in: "WARN", want: LevelWarn},
		{in: "warning", want: LevelWarn},
		{in: "error", want: LevelError},
		{in: "", want: LevelInfo},
		{in: "nope", want: LevelInfo, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseLevel(tt.in)
			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantErr, err != nil)
		})
	}
}
