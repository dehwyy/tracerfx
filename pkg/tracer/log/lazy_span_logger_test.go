package log

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestLazyFromContextAddsIDsOnce(t *testing.T) {
	var buf bytes.Buffer
	sc := spanContext(
		t,
		true,
	)
	ctx := ContextWithLogger(
		t.Context(),
		New(WithWriter(&buf)),
	)
	ctx = trace.ContextWithSpanContext(
		ctx,
		sc,
	)

	lazy := LazyFromContext(
		ctx,
		sc,
	)
	lazy.Info("direct")
	lazy.With(
		"k",
		"v",
	).Info("with")

	entries := decode(
		t,
		&buf,
	)
	require.Len(
		t,
		entries,
		2,
	)
	for _, entry := range entries {
		require.Equal(
			t,
			sc.TraceID().String(),
			entry["trace_id"],
		)
		require.Equal(
			t,
			sc.SpanID().String(),
			entry["span_id"],
		)
	}
	require.Equal(
		t,
		"v",
		entries[1]["k"],
	)
	require.Equal(
		t,
		2,
		strings.Count(
			buf.String(),
			`"trace_id"`,
		),
	)
	require.Equal(
		t,
		2,
		strings.Count(
			buf.String(),
			`"span_id"`,
		),
	)
}

func TestLazyFromContextInvalidSpanContextReturnsBase(t *testing.T) {
	var buf bytes.Buffer
	base := New(WithWriter(&buf))
	ctx := ContextWithLogger(
		t.Context(),
		base,
	)

	require.Same(
		t,
		base,
		LazyFromContext(
			ctx,
			trace.SpanContext{},
		),
	)

	LazyFromContext(
		ctx,
		trace.SpanContext{},
	).Info("msg")
	require.NotContains(
		t,
		buf.String(),
		"trace_id",
	)
}

func TestLazyFromContextIsRecognisedAsBound(t *testing.T) {
	var buf bytes.Buffer
	sc := spanContext(
		t,
		true,
	)
	ctx := trace.ContextWithSpanContext(
		t.Context(),
		sc,
	)
	ctx = ContextWithLogger(
		ctx,
		New(WithWriter(&buf)),
	)

	lazy := LazyFromContext(
		ctx,
		sc,
	)
	ctx = ContextWithLogger(
		ctx,
		lazy,
	)

	require.Same(
		t,
		lazy,
		FromContext(ctx),
	)
	require.Same(
		t,
		lazy,
		LazyFromContext(
			ctx,
			sc,
		),
	)

	FromContext(ctx).Info("msg")
	require.Equal(
		t,
		1,
		strings.Count(
			buf.String(),
			`"trace_id"`,
		),
	)
}

func TestLazyFromContextRebindsWithoutDuplicates(t *testing.T) {
	var buf bytes.Buffer
	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	require.NoError(
		t,
		err,
	)
	parentID, err := trace.SpanIDFromHex("0102030405060708")
	require.NoError(
		t,
		err,
	)
	childID, err := trace.SpanIDFromHex("1112131415161718")
	require.NoError(
		t,
		err,
	)

	parentSC := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     parentID,
		TraceFlags: trace.FlagsSampled,
	})
	childSC := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     childID,
		TraceFlags: trace.FlagsSampled,
	})

	ctx := ContextWithLogger(
		t.Context(),
		New(WithWriter(&buf)),
	)
	parentLazy := LazyFromContext(
		trace.ContextWithSpanContext(
			ctx,
			parentSC,
		),
		parentSC,
	)

	nested := ContextWithLogger(
		ctx,
		parentLazy.With(
			"k",
			"v",
		),
	)
	childLazy := LazyFromContext(
		trace.ContextWithSpanContext(
			nested,
			childSC,
		),
		childSC,
	)
	childLazy.Info("child")

	nestedLazy := ContextWithLogger(
		ctx,
		parentLazy,
	)
	LazyFromContext(
		trace.ContextWithSpanContext(
			nestedLazy,
			childSC,
		),
		childSC,
	).Info("rebound")

	require.Equal(
		t,
		2,
		strings.Count(
			buf.String(),
			`"trace_id"`,
		),
	)
	for _, entry := range decode(
		t,
		&buf,
	) {
		require.Equal(
			t,
			childID.String(),
			entry["span_id"],
		)
	}
}

func TestLazySpanLoggerLevelsAndWith(t *testing.T) {
	var buf bytes.Buffer
	sc := spanContext(
		t,
		true,
	)
	lazy := NewLazySpanLogger(
		New(
			WithWriter(&buf),
			WithLevel(LevelDebug),
		),
		sc,
	)

	lazy.Debug("d")
	lazy.Info("i")
	lazy.Warn("w")
	lazy.Error("e")

	entries := decode(
		t,
		&buf,
	)
	require.Len(
		t,
		entries,
		4,
	)
	for i, level := range []string{"debug", "info", "warn", "error"} {
		require.Equal(
			t,
			level,
			entries[i]["level"],
		)
		require.Equal(
			t,
			sc.TraceID().String(),
			entries[i]["trace_id"],
		)
	}
}

func TestLazySpanLoggerUnwrapsBoundBase(t *testing.T) {
	var buf bytes.Buffer
	sc := spanContext(
		t,
		true,
	)
	bound := newSpanLogger(
		New(WithWriter(&buf)),
		sc,
	)

	NewLazySpanLogger(
		bound,
		sc,
	).Info("msg")

	require.Equal(
		t,
		1,
		strings.Count(
			buf.String(),
			`"trace_id"`,
		),
	)
}

func TestLazySpanLoggerConcurrentResolve(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex
	sc := spanContext(
		t,
		true,
	)
	lazy := NewLazySpanLogger(
		New(WithWriter(&lockedWriter{
			mu: &mu,
			w:  &buf,
		})),
		sc,
	)

	var group sync.WaitGroup
	for i := 0; i < 32; i++ {
		group.Add(1)
		go func() {
			defer group.Done()

			lazy.Info("msg")
			lazy.With(
				"k",
				"v",
			).Warn("msg")
			DebugEnabled(lazy)
		}()
	}
	group.Wait()

	mu.Lock()
	defer mu.Unlock()
	require.Equal(
		t,
		64,
		strings.Count(
			buf.String(),
			`"trace_id"`,
		),
	)
}

type lockedWriter struct {
	mu *sync.Mutex
	w  *bytes.Buffer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.w.Write(p)
}
