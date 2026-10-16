package dspan

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestLinkCapsAtMaxLinksOnEveryDepth(t *testing.T) {
	depths := map[string]int32{
		"all":   depthAll,
		"app":   depthApp,
		"batch": depthBatch,
	}

	for name, depth := range depths {
		t.Run(
			name,
			func(t *testing.T) {
				setFast(
					t,
					true,
				)
				setDepth(
					t,
					depth,
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

				for i := 0; i < maxLinks+1; i++ {
					batch.Link(
						elementContext(
							batchCtx,
							byte(i),
							byte(i+1),
						),
					)
				}
				batch.End()

				exported := pipe.exported()
				require.Len(
					t,
					exported,
					1,
				)
				require.Len(
					t,
					exported[0].Links,
					maxLinks,
				)
			},
		)
	}
}

func TestLinkIgnoresInvalidSelfAndNilContext(t *testing.T) {
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

	var noContext context.Context
	batch.Link(noContext)
	batch.Link(ctx)
	batch.Link(batchCtx)
	batch.Link(
		trace.ContextWithRemoteSpanContext(
			ctx,
			trace.SpanContext{},
		),
	)
	batch.Link(elementContext(
		ctx,
		0x10,
		0x11,
	))

	batch.End()

	exported := pipe.exported()
	require.Len(
		t,
		exported,
		1,
	)
	require.Len(
		t,
		exported[0].Links,
		1,
	)
	require.Equal(
		t,
		foreignSpanContext(
			0x10,
			0x11,
		).SpanID(),
		exported[0].Links[0].SpanContext.SpanID(),
	)
	require.Equal(
		t,
		foreignSpanContext(
			0x10,
			0x11,
		).TraceID(),
		exported[0].Links[0].SpanContext.TraceID(),
	)
}

func TestLinkFromLightSpanLandsOnBatch(t *testing.T) {
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
	_, light := Start(
		batchCtx,
		"application.ledger.Settle",
	)
	require.True(
		t,
		light.light,
	)

	light.Link(batchCtx)
	light.Link(elementContext(
		ctx,
		0x20,
		0x21,
	))
	light.Link(elementContext(
		ctx,
		0x22,
		0x23,
	))
	light.End()

	batch.Link(elementContext(
		ctx,
		0x24,
		0x25,
	))
	batch.End()

	exported := pipe.exported()
	require.Len(
		t,
		exported,
		1,
	)
	require.Len(
		t,
		exported[0].Links,
		3,
	)
	require.Equal(
		t,
		int32(3),
		batch.links.Load(),
	)
	require.Zero(
		t,
		light.links.Load(),
	)
}

func TestLinkOnUnsampledBatchSurvivesErrorPass(t *testing.T) {
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
					"infrastructure.repository.Save",
				)
				light.Link(elementContext(
					ctx,
					0x30,
					0x31,
				))
				light.Err(errors.New("boom"))
				light.End()
				batch.End()

				exported := pipe.exported()
				require.Len(
					t,
					exported,
					1,
				)
				require.Len(
					t,
					exported[0].Links,
					1,
				)
				require.Equal(
					t,
					foreignSpanContext(
						0x30,
						0x31,
					).TraceID(),
					exported[0].Links[0].SpanContext.TraceID(),
				)
			},
		)
	}
}

func TestLinkOnRegularSpanWithoutBatch(t *testing.T) {
	setFast(
		t,
		true,
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

	_, span := Start(
		ctx,
		"application.ledger.Settle",
	)
	span.Link(elementContext(
		ctx,
		0x40,
		0x41,
	))
	span.End()

	exported := pipe.exported()
	require.Len(
		t,
		exported,
		1,
	)
	require.Len(
		t,
		exported[0].Links,
		1,
	)
}
