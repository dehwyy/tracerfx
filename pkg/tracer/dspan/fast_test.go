package dspan

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type fastInner struct {
	Value int
}

type fastPayload struct {
	Name   string
	Count  int
	Ratio  float64
	Active bool
	Inner  fastInner
	Ref    *fastInner
	hidden string
}

func newFastPayload() *fastPayload {
	return &fastPayload{
		Name:   "alice",
		Count:  3,
		Ratio:  1.5,
		Active: true,
		Inner: fastInner{
			Value: 7,
		},
		Ref: &fastInner{
			Value: 9,
		},
		hidden: "secret",
	}
}

func fastScenario(ctx context.Context) *dspan {
	_, span := Start(
		ctx,
		"application.service.Method",
		Attr(
			"start_req",
			newFastPayload(),
		),
	)
	span.WithAttribute(
		"nil_ptr",
		(*fastPayload)(nil),
	)
	span.WithAttribute(
		"plain",
		"x",
	)
	span.WithAttribute(
		"by_value",
		*newFastPayload(),
	)
	span.WithAttribute(
		"map",
		map[string]int{"a": 1},
	)
	Response(
		span,
		newFastPayload(),
	)

	return span
}

func TestFastEnabledParsesEnvOnce(t *testing.T) {
	cases := map[string]bool{
		"":       false,
		"off":    false,
		"on":     true,
		"ON":     true,
		" On ":   true,
		"yes":    false,
		"1":      false,
		"true":   false,
		"on,off": false,
	}

	for value, want := range cases {
		t.Run(
			value,
			func(t *testing.T) {
				fastState.Store(fastStateUnset)
				t.Cleanup(
					func() {
						fastState.Store(fastStateUnset)
					},
				)

				t.Setenv(
					FastEnv,
					value,
				)
				require.Equal(
					t,
					want,
					fastEnabled(),
				)

				other := "on"
				if want {
					other = "off"
				}
				t.Setenv(
					FastEnv,
					other,
				)
				require.Equal(
					t,
					want,
					fastEnabled(),
				)
			},
		)
	}
}

func TestFastSampledAttributesMatchOff(t *testing.T) {
	collect := func(fast bool) map[string]string {
		setFast(
			t,
			fast,
		)
		pipe := newPipeline(
			t,
			true,
		)
		ctx, _ := loggedContext(
			t,
			log.LevelInfo,
		)

		span := fastScenario(ctx)
		span.End()

		exported := pipe.exported()
		require.Len(
			t,
			exported,
			1,
		)

		return attributeMap(exported[0])
	}

	off := collect(false)
	on := collect(true)

	require.NotEmpty(
		t,
		off,
	)
	require.Equal(
		t,
		off,
		on,
	)
	require.Equal(
		t,
		"alice",
		on["start_req.Name"],
	)
	require.Equal(
		t,
		"7",
		on["start_req.Inner.Value"],
	)
	require.Equal(
		t,
		"9",
		on["start_req.Ref.Value"],
	)
	require.Equal(
		t,
		"<nil>",
		on["nil_ptr"],
	)
	require.Equal(
		t,
		"alice",
		on["response.Name"],
	)
	require.NotContains(
		t,
		on,
		"start_req.hidden",
	)
}

func TestFastUnsampledOkSpanHasNoAttributesAndIsNotExported(t *testing.T) {
	setFast(
		t,
		true,
	)
	pipe := newPipeline(
		t,
		false,
	)
	ctx, _ := loggedContext(
		t,
		log.LevelInfo,
	)

	span := fastScenario(ctx)
	span.End()

	require.Empty(
		t,
		pipe.exported(),
	)

	ended := pipe.recorder.Ended()
	require.Len(
		t,
		ended,
		1,
	)
	require.Empty(
		t,
		ended[0].Attributes(),
	)
	require.False(
		t,
		ended[0].SpanContext().IsSampled(),
	)
}

func TestFastUnsampledKeepsEagerAttributesWhenOff(t *testing.T) {
	setFast(
		t,
		false,
	)
	pipe := newPipeline(
		t,
		false,
	)
	ctx, _ := loggedContext(
		t,
		log.LevelInfo,
	)

	span := fastScenario(ctx)
	span.End()

	ended := pipe.recorder.Ended()
	require.Len(
		t,
		ended,
		1,
	)
	require.NotEmpty(
		t,
		ended[0].Attributes(),
	)
}

func TestFastUnsampledErrExportsSameAsOff(t *testing.T) {
	boom := errors.New("boom")

	run := func(fast bool) (tracetest.SpanStub, []map[string]any) {
		setFast(
			t,
			fast,
		)
		pipe := newPipeline(
			t,
			false,
		)
		ctx, buffer := loggedContext(
			t,
			log.LevelInfo,
		)

		span := fastScenario(ctx)
		buffer.Reset()
		require.Same(
			t,
			boom,
			span.Err(boom),
		)
		span.End()

		exported := pipe.exported()
		require.Len(
			t,
			exported,
			1,
		)

		return exported[0], logEntries(
			t,
			buffer,
		)
	}

	offSpan, offLines := run(false)
	onSpan, onLines := run(true)

	require.Equal(
		t,
		codes.Error,
		onSpan.Status.Code,
	)
	require.Equal(
		t,
		"boom",
		onSpan.Status.Description,
	)
	require.Equal(
		t,
		attributeMap(offSpan),
		attributeMap(onSpan),
	)
	require.NotEmpty(
		t,
		attributeMap(onSpan),
	)

	require.Len(
		t,
		onSpan.Events,
		1,
	)
	require.Equal(
		t,
		"exception",
		onSpan.Events[0].Name,
	)
	require.Len(
		t,
		offSpan.Events,
		1,
	)

	require.Len(
		t,
		offLines,
		1,
	)
	require.Len(
		t,
		onLines,
		1,
	)
	require.Equal(
		t,
		sortedKeys(offLines[0]),
		sortedKeys(onLines[0]),
	)
	require.Equal(
		t,
		"Span recorded error",
		onLines[0]["message"],
	)
	require.Equal(
		t,
		"boom",
		onLines[0]["error"],
	)
	require.Equal(
		t,
		"application.service.Method",
		onLines[0]["span_name"],
	)
	require.Equal(
		t,
		"alice",
		onLines[0]["start_req.Name"],
	)
	require.Equal(
		t,
		offLines[0]["response.Name"],
		onLines[0]["response.Name"],
	)
}

func TestFastWithAttributeAfterErrGoesDirect(t *testing.T) {
	setFast(
		t,
		true,
	)
	pipe := newPipeline(
		t,
		false,
	)
	ctx, _ := loggedContext(
		t,
		log.LevelInfo,
	)

	_, span := Start(
		ctx,
		"op",
		Attr(
			"before",
			"a",
		),
	)
	span.Err(errors.New("boom"))
	span.WithAttribute(
		"late",
		"b",
	)
	Response(
		span,
		"done",
	)
	span.End()

	exported := pipe.exported()
	require.Len(
		t,
		exported,
		1,
	)
	require.Equal(
		t,
		map[string]string{
			"before":   "a",
			"late":     "b",
			"response": "done",
		},
		attributeMap(exported[0]),
	)
}

func TestFastExpectedErrorUnsampledOnlyLogsAtDebug(t *testing.T) {
	expected := errors.New("expected")
	ResetExpectedErrors()
	RegisterExpectedErrors(
		func(err error) bool {
			return errors.Is(
				err,
				expected,
			)
		},
	)
	expectedState.Store(expectedStateUnset)
	t.Cleanup(
		func() {
			ResetExpectedErrors()
			expectedState.Store(expectedStateUnset)
		},
	)

	setFast(
		t,
		true,
	)

	t.Run(
		"info level is silent",
		func(t *testing.T) {
			pipe := newPipeline(
				t,
				false,
			)
			ctx, buffer := loggedContext(
				t,
				log.LevelInfo,
			)

			_, span := Start(
				ctx,
				"op",
				Attr(
					"k",
					"v",
				),
			)
			require.Same(
				t,
				expected,
				span.Err(expected),
			)
			span.End()

			require.Empty(
				t,
				logEntries(
					t,
					buffer,
				),
			)
			require.Empty(
				t,
				pipe.exported(),
			)

			ended := pipe.recorder.Ended()
			require.Len(
				t,
				ended,
				1,
			)
			require.Equal(
				t,
				codes.Unset,
				ended[0].Status().Code,
			)
			require.Empty(
				t,
				ended[0].Attributes(),
			)
			require.Empty(
				t,
				ended[0].Events(),
			)
		},
	)

	t.Run(
		"debug level logs attributes",
		func(t *testing.T) {
			pipe := newPipeline(
				t,
				false,
			)
			ctx, buffer := loggedContext(
				t,
				log.LevelDebug,
			)

			_, span := Start(
				ctx,
				"op",
				Attr(
					"k",
					"v",
				),
			)
			buffer.Reset()
			span.Err(expected)

			lines := logEntries(
				t,
				buffer,
			)
			require.Len(
				t,
				lines,
				1,
			)
			require.Equal(
				t,
				"Span recorded expected error",
				lines[0]["message"],
			)
			require.Equal(
				t,
				"v",
				lines[0]["k"],
			)
			require.Equal(
				t,
				"debug",
				lines[0]["level"],
			)

			span.End()
			require.Empty(
				t,
				pipe.exported(),
			)
		},
	)

	t.Run(
		"sampled span keeps expected attribute",
		func(t *testing.T) {
			pipe := newPipeline(
				t,
				true,
			)
			ctx, _ := loggedContext(
				t,
				log.LevelInfo,
			)

			_, span := Start(
				ctx,
				"op",
			)
			span.Err(expected)
			span.End()

			exported := pipe.exported()
			require.Len(
				t,
				exported,
				1,
			)
			require.Equal(
				t,
				codes.Unset,
				exported[0].Status.Code,
			)
			require.Equal(
				t,
				"true",
				attributeMap(exported[0])[expectedErrorAttribute],
			)
		},
	)
}

func TestFastSpanLoggerCarriesTraceIDOnce(t *testing.T) {
	setFast(
		t,
		true,
	)
	newPipeline(
		t,
		true,
	)

	ctx, buffer := loggedContext(
		t,
		log.LevelDebug,
	)

	ctxParent, parent := Start(
		ctx,
		"parent",
	)
	parent.Logger.Warn("from parent")

	lines := logEntries(
		t,
		buffer,
	)
	last := lines[len(lines)-1]
	require.Equal(
		t,
		"from parent",
		last["message"],
	)
	require.Equal(
		t,
		parent.TraceID(),
		last["trace_id"],
	)
	require.Equal(
		t,
		parent.TraceSpan.SpanContext().SpanID().String(),
		last["span_id"],
	)
	require.Equal(
		t,
		1,
		countKey(
			buffer.String(),
			"from parent",
			`"trace_id"`,
		),
	)
	require.Equal(
		t,
		1,
		countKey(
			buffer.String(),
			"from parent",
			`"span_id"`,
		),
	)

	nested := log.ContextWithLogger(
		ctxParent,
		parent.Logger,
	)
	_, child := Start(
		nested,
		"child",
	)
	buffer.Reset()
	child.Logger.Warn("from child")

	lines = logEntries(
		t,
		buffer,
	)
	require.Len(
		t,
		lines,
		1,
	)
	require.Equal(
		t,
		child.TraceID(),
		lines[0]["trace_id"],
	)
	require.Equal(
		t,
		parent.TraceID(),
		lines[0]["trace_id"],
	)
	require.Equal(
		t,
		child.TraceSpan.SpanContext().SpanID().String(),
		lines[0]["span_id"],
	)
	require.NotEqual(
		t,
		parent.TraceSpan.SpanContext().SpanID().String(),
		lines[0]["span_id"],
	)
	require.Equal(
		t,
		1,
		countKey(
			buffer.String(),
			"",
			`"trace_id"`,
		),
	)
	require.Equal(
		t,
		1,
		countKey(
			buffer.String(),
			"",
			`"span_id"`,
		),
	)

	buffer.Reset()
	child.End()
	parent.End()
	for _, line := range logEntries(
		t,
		buffer,
	) {
		require.Equal(
			t,
			"Span ended",
			line["message"],
		)
	}
	require.Equal(
		t,
		2,
		countKey(
			buffer.String(),
			"",
			`"trace_id"`,
		),
	)
}

func TestFastTracerCacheFollowsProvider(t *testing.T) {
	setFast(
		t,
		true,
	)

	first := newPipeline(
		t,
		true,
	)
	ctx, _ := loggedContext(
		t,
		log.LevelInfo,
	)

	_, span := Start(
		ctx,
		"first",
	)
	span.End()

	second := newPipeline(
		t,
		true,
	)

	_, span = Start(
		ctx,
		"second",
	)
	span.End()

	require.Equal(
		t,
		[]string{"first"},
		first.exportedNames(),
	)
	require.Equal(
		t,
		[]string{"second"},
		second.exportedNames(),
	)
}

func TestSpanCallerOnlyWhenDebugEnabledInFastMode(t *testing.T) {
	newPipeline(
		t,
		true,
	)

	t.Run(
		"fast debug",
		func(t *testing.T) {
			setFast(
				t,
				true,
			)
			ctx, buffer := loggedContext(
				t,
				log.LevelDebug,
			)

			_, span := Start(
				ctx,
				"op",
			)
			require.Contains(
				t,
				span.SpanCaller,
				"TestSpanCallerOnlyWhenDebugEnabledInFastMode",
			)
			require.Contains(
				t,
				buffer.String(),
				"TestSpanCallerOnlyWhenDebugEnabledInFastMode",
			)
		},
	)

	t.Run(
		"fast info",
		func(t *testing.T) {
			setFast(
				t,
				true,
			)
			ctx, buffer := loggedContext(
				t,
				log.LevelInfo,
			)

			_, span := Start(
				ctx,
				"op",
			)
			require.Empty(
				t,
				span.SpanCaller,
			)
			span.End()
			require.Empty(
				t,
				buffer.String(),
			)
		},
	)

	t.Run(
		"off info",
		func(t *testing.T) {
			setFast(
				t,
				false,
			)
			ctx, _ := loggedContext(
				t,
				log.LevelInfo,
			)

			_, span := Start(
				ctx,
				"op",
			)
			require.Contains(
				t,
				span.SpanCaller,
				"TestSpanCallerOnlyWhenDebugEnabledInFastMode",
			)
		},
	)
}

func TestFastDebugEndLineCarriesPendingAttributes(t *testing.T) {
	setFast(
		t,
		true,
	)
	newPipeline(
		t,
		false,
	)
	ctx, buffer := loggedContext(
		t,
		log.LevelDebug,
	)

	_, span := Start(
		ctx,
		"op",
		Attr(
			"req",
			newFastPayload(),
		),
	)
	Response(
		span,
		"ok",
	)
	buffer.Reset()
	span.End()

	lines := logEntries(
		t,
		buffer,
	)
	require.Len(
		t,
		lines,
		1,
	)
	require.Equal(
		t,
		"Span ended",
		lines[0]["message"],
	)
	require.Equal(
		t,
		"alice",
		lines[0]["req.Name"],
	)
	require.Equal(
		t,
		"ok",
		lines[0]["response"],
	)
}

type countingLogger struct {
	mu      sync.Mutex
	calls   map[string]int
	debugOn bool
}

func newCountingLogger(debugOn bool) *countingLogger {
	return &countingLogger{
		calls:   map[string]int{},
		debugOn: debugOn,
	}
}

func (c *countingLogger) count(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.calls[name]++
}

func (c *countingLogger) total(name string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.calls[name]
}

func (c *countingLogger) Debug(string, ...any) { c.count("Debug") }
func (c *countingLogger) Info(string, ...any)  { c.count("Info") }
func (c *countingLogger) Warn(string, ...any)  { c.count("Warn") }
func (c *countingLogger) Error(string, ...any) { c.count("Error") }
func (c *countingLogger) With(...any) log.Logger {
	c.count("With")

	return c
}

type levelAwareLogger struct {
	*countingLogger
}

func (l levelAwareLogger) DebugEnabled() bool {
	return l.debugOn
}

func TestFastDisabledDebugDoesNotTouchLogger(t *testing.T) {
	setFast(
		t,
		true,
	)
	newPipeline(
		t,
		true,
	)

	counter := newCountingLogger(false)
	ctx := log.ContextWithLogger(
		t.Context(),
		levelAwareLogger{
			countingLogger: counter,
		},
	)

	_, span := Start(
		ctx,
		"op",
		Attr(
			"req",
			newFastPayload(),
		),
	)
	span.WithAttribute(
		"k",
		"v",
	)
	Response(
		span,
		"ok",
	)
	span.End()

	require.Zero(
		t,
		counter.total("With"),
	)
	require.Zero(
		t,
		counter.total("Debug"),
	)
}

func TestFastForeignLoggerKeepsDebug(t *testing.T) {
	setFast(
		t,
		true,
	)
	newPipeline(
		t,
		true,
	)

	counter := newCountingLogger(false)
	ctx := log.ContextWithLogger(
		t.Context(),
		counter,
	)

	_, span := Start(
		ctx,
		"op",
	)
	span.End()

	require.Equal(
		t,
		2,
		counter.total("Debug"),
	)
	require.Equal(
		t,
		1,
		counter.total("With"),
	)
}

func TestFastOffMatchesLegacyLoggerUse(t *testing.T) {
	setFast(
		t,
		false,
	)
	newPipeline(
		t,
		true,
	)

	counter := newCountingLogger(false)
	ctx := log.ContextWithLogger(
		t.Context(),
		levelAwareLogger{
			countingLogger: counter,
		},
	)

	_, span := Start(
		ctx,
		"op",
	)
	span.End()

	require.Equal(
		t,
		2,
		counter.total("Debug"),
	)
	require.Equal(
		t,
		1,
		counter.total("With"),
	)
}

func TestFastConcurrentAttributesAndErr(t *testing.T) {
	setFast(
		t,
		true,
	)
	pipe := newPipeline(
		t,
		false,
	)
	ctx := log.ContextWithLogger(
		t.Context(),
		log.New(
			log.WithWriter(io.Discard),
		),
	)

	_, span := Start(
		ctx,
		"op",
	)

	var group sync.WaitGroup
	for i := 0; i < 32; i++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()

			span.WithAttribute(
				"key",
				index,
			)
			if index%8 == 0 {
				span.Err(errors.New("boom"))
			}
			span.Logger.Warn("concurrent")
		}(i)
	}
	group.Wait()
	span.End()

	exported := pipe.exported()
	require.Len(
		t,
		exported,
		1,
	)
	require.Equal(
		t,
		codes.Error,
		exported[0].Status.Code,
	)
	require.Contains(
		t,
		attributeMap(exported[0]),
		"key",
	)
}

func TestFastTracerCacheIgnoresUncomparableProvider(t *testing.T) {
	setFast(
		t,
		true,
	)

	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(uncomparableProvider{
		TracerProvider: tracesdk.NewTracerProvider(),
		marker:         []int{1},
	})
	t.Cleanup(
		func() {
			otel.SetTracerProvider(previous)
		},
	)

	ctx, _ := loggedContext(
		t,
		log.LevelInfo,
	)
	for i := 0; i < 3; i++ {
		_, span := Start(
			ctx,
			"op",
		)
		span.End()
	}
}

type uncomparableProvider struct {
	trace.TracerProvider
	marker []int
}

func countKey(output string, message string, key string) int {
	count := 0
	for _, line := range strings.Split(
		output,
		"\n",
	) {
		if message != "" && !strings.Contains(
			line,
			message,
		) {
			continue
		}

		count += strings.Count(
			line,
			key,
		)
	}

	return count
}
