package dspan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func setDepth(t *testing.T, depth int32) {
	t.Helper()

	depthState.Store(depth)
	t.Cleanup(
		func() {
			depthState.Store(depthUnset)
		},
	)
}

func spanByName(t *testing.T, stubs tracetest.SpanStubs, name string) tracetest.SpanStub {
	t.Helper()

	for _, stub := range stubs {
		if stub.Name == name {
			return stub
		}
	}

	require.FailNow(
		t,
		"span not exported",
		name,
	)

	return tracetest.SpanStub{}
}

func foreignSpanContext(traceByte byte, spanByte byte) trace.SpanContext {
	return trace.NewSpanContext(
		trace.SpanContextConfig{
			TraceID:    trace.TraceID{traceByte, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			SpanID:     trace.SpanID{spanByte, 1, 2, 3, 4, 5, 6, 7},
			TraceFlags: trace.FlagsSampled,
		},
	)
}

func elementContext(ctx context.Context, traceByte byte, spanByte byte) context.Context {
	return trace.ContextWithRemoteSpanContext(
		ctx,
		foreignSpanContext(
			traceByte,
			spanByte,
		),
	)
}

func nestedWorkload(ctx context.Context) {
	ctx, app := Start(
		ctx,
		"application.ledger.Settle",
		Attr(
			"order",
			"o-1",
		),
	)
	defer app.End()

	_, repo := Start(
		ctx,
		"infrastructure.repository.Save",
		Attr(
			"table",
			"postings",
		),
	)
	repo.End()

	_, gateway := Start(
		ctx,
		"infrastructure.gateway.Publish",
	)
	gateway.End()
}

func TestParseSpanDepth(t *testing.T) {
	tests := []struct {
		raw       string
		wantDepth int32
		wantValid bool
	}{
		{
			raw:       "",
			wantDepth: depthAll,
			wantValid: true,
		},
		{
			raw:       "all",
			wantDepth: depthAll,
			wantValid: true,
		},
		{
			raw:       "app",
			wantDepth: depthApp,
			wantValid: true,
		},
		{
			raw:       "batch",
			wantDepth: depthBatch,
			wantValid: true,
		},
		{
			raw:       " BATCH ",
			wantDepth: depthBatch,
			wantValid: true,
		},
		{
			raw:       "App",
			wantDepth: depthApp,
			wantValid: true,
		},
		{
			raw:       "minimal",
			wantDepth: depthAll,
			wantValid: false,
		},
		{
			raw:       "bogus",
			wantDepth: depthAll,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.raw,
			func(t *testing.T) {
				depth, valid := parseSpanDepth(tt.raw)

				require.Equal(
					t,
					tt.wantDepth,
					depth,
				)
				require.Equal(
					t,
					tt.wantValid,
					valid,
				)
			},
		)
	}
}

func TestSpanDepthReadOnceWarnsOnce(t *testing.T) {
	buffer := &bytes.Buffer{}
	log.SetDefaultLogger(
		log.New(
			log.WithWriter(buffer),
		),
	)
	t.Cleanup(
		func() {
			log.SetDefaultLogger(nil)
			depthState.Store(depthUnset)
		},
	)

	depthState.Store(depthUnset)
	t.Setenv(
		SpanDepthEnv,
		"minimal",
	)

	require.Equal(
		t,
		depthAll,
		spanDepth(),
	)

	t.Setenv(
		SpanDepthEnv,
		"batch",
	)
	require.Equal(
		t,
		depthAll,
		spanDepth(),
	)
	require.Equal(
		t,
		depthAll,
		spanDepth(),
	)

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
		"warn",
		lines[0]["level"],
	)
	require.Equal(
		t,
		"minimal",
		lines[0]["value"],
	)
	require.Equal(
		t,
		SpanDepthEnv,
		lines[0]["env"],
	)
}

func TestSpanDepthValidEnvDoesNotWarn(t *testing.T) {
	buffer := &bytes.Buffer{}
	log.SetDefaultLogger(
		log.New(
			log.WithWriter(buffer),
		),
	)
	t.Cleanup(
		func() {
			log.SetDefaultLogger(nil)
			depthState.Store(depthUnset)
		},
	)

	depthState.Store(depthUnset)
	t.Setenv(
		SpanDepthEnv,
		"batch",
	)

	require.Equal(
		t,
		depthBatch,
		spanDepth(),
	)
	require.Empty(
		t,
		buffer.String(),
	)
}

func TestStartBatchAtAllMatchesStart(t *testing.T) {
	for _, fast := range []bool{false, true} {
		t.Run(
			fmt.Sprintf(
				"fast=%v",
				fast,
			),
			func(t *testing.T) {
				setFast(
					t,
					fast,
				)
				setDepth(
					t,
					depthAll,
				)
				pipe := newPipeline(
					t,
					true,
				)
				ctx, _ := loggedContext(
					t,
					log.LevelInfo,
				)

				batchCtx, batch := StartBatch(
					ctx,
					"batch.process",
				)
				require.Nil(
					t,
					activeScope(batchCtx),
				)

				nestedWorkload(batchCtx)
				batch.End()

				exported := pipe.exported()
				require.Len(
					t,
					exported,
					4,
				)

				root := spanByName(
					t,
					exported,
					"batch.process",
				)
				app := spanByName(
					t,
					exported,
					"application.ledger.Settle",
				)
				repo := spanByName(
					t,
					exported,
					"infrastructure.repository.Save",
				)

				require.Equal(
					t,
					root.SpanContext.SpanID(),
					app.Parent.SpanID(),
				)
				require.Equal(
					t,
					app.SpanContext.SpanID(),
					repo.Parent.SpanID(),
				)
				require.NotContains(
					t,
					attributeMap(root),
					batchFailedAttribute,
				)
				require.NotContains(
					t,
					attributeMap(root),
					batchSuppressedAttribute,
				)
			},
		)
	}
}

func TestStartBatchAtAppSuppressesOnlyInfrastructure(t *testing.T) {
	for _, fast := range []bool{false, true} {
		t.Run(
			fmt.Sprintf(
				"fast=%v",
				fast,
			),
			func(t *testing.T) {
				setFast(
					t,
					fast,
				)
				setDepth(
					t,
					depthApp,
				)
				pipe := newPipeline(
					t,
					true,
				)
				ctx, _ := loggedContext(
					t,
					log.LevelInfo,
				)

				batchCtx, batch := StartBatch(
					ctx,
					"batch.process",
				)
				require.NotNil(
					t,
					activeScope(batchCtx),
				)

				nestedWorkload(batchCtx)
				batch.End()

				exported := pipe.exported()
				require.ElementsMatch(
					t,
					[]string{"batch.process", "application.ledger.Settle"},
					pipe.exportedNames(),
				)

				root := spanByName(
					t,
					exported,
					"batch.process",
				)
				app := spanByName(
					t,
					exported,
					"application.ledger.Settle",
				)

				require.Equal(
					t,
					root.SpanContext.SpanID(),
					app.Parent.SpanID(),
				)
				require.Equal(
					t,
					"2",
					attributeMap(root)[batchSuppressedAttribute],
				)
				require.Equal(
					t,
					"false",
					attributeMap(root)[batchFailedAttribute],
				)
			},
		)
	}
}

func TestStartBatchAtBatchKeepsOneSpan(t *testing.T) {
	for _, fast := range []bool{false, true} {
		for _, sampled := range []bool{true, false} {
			t.Run(
				fmt.Sprintf(
					"fast=%v/sampled=%v",
					fast,
					sampled,
				),
				func(t *testing.T) {
					setFast(
						t,
						fast,
					)
					setDepth(
						t,
						depthBatch,
					)
					pipe := newPipeline(
						t,
						sampled,
					)
					ctx, _ := loggedContext(
						t,
						log.LevelInfo,
					)

					batchCtx, batch := StartBatch(
						ctx,
						"batch.process",
					)
					nestedWorkload(batchCtx)
					batch.End()

					if !sampled {
						require.Empty(
							t,
							pipe.exported(),
						)
						require.Len(
							t,
							pipe.recorder.Ended(),
							1,
						)

						return
					}

					exported := pipe.exported()
					require.Len(
						t,
						exported,
						1,
					)
					require.Equal(
						t,
						"batch.process",
						exported[0].Name,
					)
					require.Equal(
						t,
						"3",
						attributeMap(exported[0])[batchSuppressedAttribute],
					)
					require.Equal(
						t,
						"false",
						attributeMap(exported[0])[batchFailedAttribute],
					)
				},
			)
		}
	}
}

func TestLightSpanIdentity(t *testing.T) {
	setFast(
		t,
		true,
	)
	setDepth(
		t,
		depthBatch,
	)
	newPipeline(
		t,
		true,
	)
	ctx, buffer := loggedContext(
		t,
		log.LevelInfo,
	)

	batchCtx, batch := StartBatch(
		ctx,
		"batch.process",
	)

	lightCtx, light := Start(
		batchCtx,
		"application.ledger.Settle",
		Attr(
			"order",
			"o-1",
		),
	)

	require.True(
		t,
		batchCtx == lightCtx,
	)
	require.False(
		t,
		light.TraceSpan.IsRecording(),
	)
	require.Equal(
		t,
		batch.TraceSpan.SpanContext(),
		light.TraceSpan.SpanContext(),
	)
	require.Equal(
		t,
		batch.TraceID(),
		light.TraceID(),
	)
	require.Nil(
		t,
		light.Attributes,
	)

	light.Logger.Warn("from light")

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
		batch.TraceID(),
		lines[0]["trace_id"],
	)
	require.Equal(
		t,
		batch.TraceSpan.SpanContext().SpanID().String(),
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

	light.SetName("renamed")
	require.Equal(
		t,
		"renamed",
		light.SpanName,
	)

	light.End()
	batch.End()
}

func TestLightSpanOffPathDoesNotSetFast(t *testing.T) {
	setFast(
		t,
		false,
	)
	setDepth(
		t,
		depthBatch,
	)
	newPipeline(
		t,
		true,
	)
	ctx, _ := loggedContext(
		t,
		log.LevelInfo,
	)

	batchCtx, batch := StartBatch(
		ctx,
		"batch.process",
	)
	require.NotNil(
		t,
		batch.Attributes,
	)

	_, light := Start(
		batchCtx,
		"application.ledger.Settle",
	)
	require.True(
		t,
		light.light,
	)
	require.Empty(
		t,
		light.SpanCaller,
	)
}

func TestLightErrorMarksBatchFailed(t *testing.T) {
	boom := errors.New("boom")

	for _, fast := range []bool{false, true} {
		t.Run(
			fmt.Sprintf(
				"fast=%v",
				fast,
			),
			func(t *testing.T) {
				setFast(
					t,
					fast,
				)
				setDepth(
					t,
					depthBatch,
				)
				pipe := newPipeline(
					t,
					false,
				)
				ctx, buffer := loggedContext(
					t,
					log.LevelInfo,
				)

				batchCtx, batch := StartBatch(
					ctx,
					"batch.process",
					Attr(
						"batch_size",
						5,
					),
				)
				batch.WithAttribute(
					"source",
					"stream",
				)

				lightCtx, light := Start(
					batchCtx,
					"infrastructure.repository.Save",
					Attr(
						"table",
						"postings",
					),
				)
				Response(
					light,
					"partial",
				)

				require.Same(
					t,
					boom,
					light.Err(boom),
				)
				light.End()

				_, second := Start(
					lightCtx,
					"application.ledger.Settle",
				)
				require.Error(
					t,
					second.Err(errors.New("again")),
				)
				second.End()

				batch.End()

				exported := pipe.exported()
				require.Len(
					t,
					exported,
					1,
				)
				require.Equal(
					t,
					"batch.process",
					exported[0].Name,
				)
				require.Equal(
					t,
					codes.Error,
					exported[0].Status.Code,
				)
				require.Equal(
					t,
					"boom",
					exported[0].Status.Description,
				)
				require.Len(
					t,
					exported[0].Events,
					2,
				)
				require.Equal(
					t,
					"exception",
					exported[0].Events[0].Name,
				)

				attributes := attributeMap(exported[0])
				require.Equal(
					t,
					"5",
					attributes["batch_size"],
				)
				require.Equal(
					t,
					"stream",
					attributes["source"],
				)
				require.Equal(
					t,
					"true",
					attributes[batchFailedAttribute],
				)
				require.Equal(
					t,
					"2",
					attributes[batchSuppressedAttribute],
				)
				require.NotContains(
					t,
					attributes,
					"table",
				)

				lines := logEntries(
					t,
					buffer,
				)
				require.Len(
					t,
					lines,
					2,
				)
				require.Equal(
					t,
					"Span recorded error",
					lines[0]["message"],
				)
				require.Equal(
					t,
					"infrastructure.repository.Save",
					lines[0]["span_name"],
				)
				require.Equal(
					t,
					"boom",
					lines[0]["error"],
				)
				require.Equal(
					t,
					"postings",
					lines[0]["table"],
				)
				require.Equal(
					t,
					"partial",
					lines[0]["response"],
				)
				require.Equal(
					t,
					"application.ledger.Settle",
					lines[1]["span_name"],
				)
				require.Equal(
					t,
					batch.TraceID(),
					lines[0]["trace_id"],
				)
			},
		)
	}
}

func TestLightErrorKeepsErrorLineKeysOfFullSpan(t *testing.T) {
	boom := errors.New("boom")

	run := func(depth int32) map[string]any {
		setFast(
			t,
			true,
		)
		setDepth(
			t,
			depth,
		)
		newPipeline(
			t,
			false,
		)
		ctx, buffer := loggedContext(
			t,
			log.LevelInfo,
		)

		batchCtx, batch := StartBatch(
			ctx,
			"batch.process",
		)
		_, span := Start(
			batchCtx,
			"infrastructure.repository.Save",
			Attr(
				"req",
				newFastPayload(),
			),
		)
		buffer.Reset()
		span.Err(boom)
		span.End()
		batch.End()

		lines := logEntries(
			t,
			buffer,
		)
		require.Len(
			t,
			lines,
			1,
		)

		return lines[0]
	}

	full := run(depthAll)
	light := run(depthBatch)

	require.Equal(
		t,
		sortedKeys(full),
		sortedKeys(light),
	)
	require.Equal(
		t,
		full["span_name"],
		light["span_name"],
	)
	require.Equal(
		t,
		full["req.Name"],
		light["req.Name"],
	)
}

func TestLightErrorSampledBatchGetsError(t *testing.T) {
	setFast(
		t,
		false,
	)
	setDepth(
		t,
		depthBatch,
	)
	pipe := newPipeline(
		t,
		true,
	)
	ctx, _ := loggedContext(
		t,
		log.LevelInfo,
	)

	batchCtx, batch := StartBatch(
		ctx,
		"batch.process",
	)
	_, light := Start(
		batchCtx,
		"application.ledger.Settle",
	)
	light.Err(errors.New("boom"))
	light.End()
	batch.End()

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
}

func TestLightExpectedErrorOnlyLogsAtDebug(t *testing.T) {
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

	setDepth(
		t,
		depthBatch,
	)

	t.Run(
		"info",
		func(t *testing.T) {
			pipe := newPipeline(
				t,
				false,
			)
			ctx, buffer := loggedContext(
				t,
				log.LevelInfo,
			)

			batchCtx, batch := StartBatch(
				ctx,
				"batch.process",
			)
			_, light := Start(
				batchCtx,
				"application.ledger.Settle",
			)
			light.Err(expected)
			light.End()
			batch.End()

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
			require.False(
				t,
				batch.scope.failed.Load(),
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
				ended[0].Events(),
			)
		},
	)

	t.Run(
		"debug",
		func(t *testing.T) {
			newPipeline(
				t,
				false,
			)
			ctx, buffer := loggedContext(
				t,
				log.LevelDebug,
			)

			batchCtx, batch := StartBatch(
				ctx,
				"batch.process",
			)
			_, light := Start(
				batchCtx,
				"application.ledger.Settle",
			)
			buffer.Reset()
			light.Err(expected)

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
				"application.ledger.Settle",
				lines[0]["span_name"],
			)
			require.False(
				t,
				batch.scope.failed.Load(),
			)
		},
	)
}

func TestBatchOwnErrMarksFailed(t *testing.T) {
	setFast(
		t,
		true,
	)
	setDepth(
		t,
		depthBatch,
	)
	pipe := newPipeline(
		t,
		false,
	)
	ctx, _ := loggedContext(
		t,
		log.LevelInfo,
	)

	_, batch := StartBatch(
		ctx,
		"batch.process",
		Attr(
			"batch_size",
			3,
		),
	)
	batch.Err(errors.New("batch failed"))
	batch.End()

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
	require.Equal(
		t,
		"true",
		attributeMap(exported[0])[batchFailedAttribute],
	)
	require.Equal(
		t,
		"3",
		attributeMap(exported[0])["batch_size"],
	)
}

func TestTraceChangeDisablesScope(t *testing.T) {
	setFast(
		t,
		true,
	)
	setDepth(
		t,
		depthBatch,
	)
	pipe := newPipeline(
		t,
		true,
	)
	ctx, _ := loggedContext(
		t,
		log.LevelInfo,
	)

	batchCtx, batch := StartBatch(
		ctx,
		"batch.process",
	)

	orderCtx := elementContext(
		batchCtx,
		0xaa,
		0xbb,
	)
	require.Nil(
		t,
		activeScope(orderCtx),
	)

	_, order := Start(
		orderCtx,
		"application.order.Handle",
	)
	require.False(
		t,
		order.light,
	)
	require.True(
		t,
		order.TraceSpan.IsRecording(),
	)
	require.Equal(
		t,
		foreignSpanContext(
			0xaa,
			0xbb,
		).TraceID().String(),
		order.TraceID(),
	)
	order.End()

	_, inBatch := Start(
		batchCtx,
		"application.ledger.Settle",
	)
	require.True(
		t,
		inBatch.light,
	)
	inBatch.End()

	batch.End()

	exported := pipe.exported()
	handled := spanByName(
		t,
		exported,
		"application.order.Handle",
	)
	require.Equal(
		t,
		foreignSpanContext(
			0xaa,
			0xbb,
		).SpanID(),
		handled.Parent.SpanID(),
	)
	require.Equal(
		t,
		"1",
		attributeMap(spanByName(
			t,
			exported,
			"batch.process",
		))[batchSuppressedAttribute],
	)
}

func TestNestedStartBatchBehavesAsStart(t *testing.T) {
	setFast(
		t,
		true,
	)

	t.Run(
		"batch depth",
		func(t *testing.T) {
			setDepth(
				t,
				depthBatch,
			)
			pipe := newPipeline(
				t,
				true,
			)
			ctx, _ := loggedContext(
				t,
				log.LevelInfo,
			)

			batchCtx, outer := StartBatch(
				ctx,
				"batch.outer",
			)
			innerCtx, inner := StartBatch(
				batchCtx,
				"batch.inner",
			)

			require.True(
				t,
				inner.light,
			)
			require.True(
				t,
				innerCtx == batchCtx,
			)
			require.Same(
				t,
				outer.scope,
				activeScope(innerCtx),
			)

			inner.End()
			outer.End()

			require.Equal(
				t,
				[]string{"batch.outer"},
				pipe.exportedNames(),
			)
			require.Equal(
				t,
				"1",
				attributeMap(pipe.exported()[0])[batchSuppressedAttribute],
			)
		},
	)

	t.Run(
		"app depth",
		func(t *testing.T) {
			setDepth(
				t,
				depthApp,
			)
			pipe := newPipeline(
				t,
				true,
			)
			ctx, _ := loggedContext(
				t,
				log.LevelInfo,
			)

			batchCtx, outer := StartBatch(
				ctx,
				"batch.outer",
			)
			appCtx, appSpan := StartBatch(
				batchCtx,
				"application.ledger.Settle",
			)
			require.False(
				t,
				appSpan.light,
			)
			require.Nil(
				t,
				appSpan.scope,
			)
			require.Same(
				t,
				outer.scope,
				activeScope(appCtx),
			)

			_, infra := StartBatch(
				appCtx,
				"infrastructure.repository.Save",
			)
			require.True(
				t,
				infra.light,
			)

			infra.End()
			appSpan.End()
			outer.End()

			require.ElementsMatch(
				t,
				[]string{"batch.outer", "application.ledger.Settle"},
				pipe.exportedNames(),
			)
		},
	)

	t.Run(
		"foreign trace starts new scope",
		func(t *testing.T) {
			setDepth(
				t,
				depthBatch,
			)
			pipe := newPipeline(
				t,
				true,
			)
			ctx, _ := loggedContext(
				t,
				log.LevelInfo,
			)

			batchCtx, outer := StartBatch(
				ctx,
				"batch.outer",
			)
			orderCtx := elementContext(
				batchCtx,
				0xcc,
				0xdd,
			)
			nestedCtx, nested := StartBatch(
				orderCtx,
				"batch.nested",
			)

			require.False(
				t,
				nested.light,
			)
			require.NotNil(
				t,
				nested.scope,
			)
			require.NotSame(
				t,
				outer.scope,
				nested.scope,
			)
			require.Same(
				t,
				nested.scope,
				activeScope(nestedCtx),
			)

			nested.End()
			outer.End()

			require.ElementsMatch(
				t,
				[]string{"batch.outer", "batch.nested"},
				pipe.exportedNames(),
			)
		},
	)
}

func TestStartBatchWithoutRealProviderIsPlainSpan(t *testing.T) {
	setDepth(
		t,
		depthBatch,
	)

	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(noop.NewTracerProvider())
	t.Cleanup(
		func() {
			otel.SetTracerProvider(previous)
		},
	)

	ctx, _ := loggedContext(
		t,
		log.LevelInfo,
	)

	batchCtx, batch := StartBatch(
		ctx,
		"batch.process",
	)
	require.Nil(
		t,
		batch.scope,
	)
	require.Nil(
		t,
		activeScope(batchCtx),
	)

	_, child := Start(
		batchCtx,
		"application.ledger.Settle",
	)
	require.False(
		t,
		child.light,
	)

	child.End()
	batch.End()
}

func TestStartBatchSpanCaller(t *testing.T) {
	newPipeline(
		t,
		true,
	)

	for _, fast := range []bool{false, true} {
		t.Run(
			fmt.Sprintf(
				"fast=%v",
				fast,
			),
			func(t *testing.T) {
				setFast(
					t,
					fast,
				)
				setDepth(
					t,
					depthBatch,
				)
				ctx, _ := loggedContext(
					t,
					log.LevelDebug,
				)

				batchCtx, batch := StartBatch(
					ctx,
					"batch.process",
				)
				require.Contains(
					t,
					batch.SpanCaller,
					"TestStartBatchSpanCaller",
				)

				_, light := Start(
					batchCtx,
					"application.ledger.Settle",
				)
				require.Contains(
					t,
					light.SpanCaller,
					"TestStartBatchSpanCaller",
				)

				_, nested := StartBatch(
					batchCtx,
					"batch.nested",
				)
				require.Contains(
					t,
					nested.SpanCaller,
					"TestStartBatchSpanCaller",
				)
			},
		)
	}
}

func TestLightSpansConcurrentErr(t *testing.T) {
	setFast(
		t,
		true,
	)
	setDepth(
		t,
		depthBatch,
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

	batchCtx, batch := StartBatch(
		ctx,
		"batch.process",
		Attr(
			"batch_size",
			200,
		),
	)

	const workers = 200

	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()

			_, light := Start(
				batchCtx,
				"infrastructure.repository.Save",
				Attr(
					"index",
					index,
				),
			)
			if index%2 == 0 {
				light.Err(fmt.Errorf(
					"boom %d",
					index,
				))
			}
			light.Link(elementContext(
				batchCtx,
				byte(index),
				byte(index),
			))
			light.End()
		}(i)
	}
	group.Wait()
	batch.End()

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
	require.LessOrEqual(
		t,
		len(exported[0].Events),
		128,
	)
	require.Equal(
		t,
		"200",
		attributeMap(exported[0])[batchSuppressedAttribute],
	)
	require.Equal(
		t,
		"true",
		attributeMap(exported[0])[batchFailedAttribute],
	)
	require.Len(
		t,
		exported[0].Links,
		maxLinks,
	)
}
