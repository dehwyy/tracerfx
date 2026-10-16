package dspan

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

var errExpectedNotFound = errors.New("expected not found")

func isExpectedNotFound(err error) bool {
	return errors.Is(
		err,
		errExpectedNotFound,
	)
}

type expectedHarness struct {
	recorder *tracetest.SpanRecorder
	buffer   *bytes.Buffer
	span     *dspan
}

func newExpectedHarness(t *testing.T, envValue string) *expectedHarness {
	t.Helper()

	previous := otel.GetTracerProvider()
	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(
		tracesdk.NewTracerProvider(
			tracesdk.WithSpanProcessor(recorder),
		),
	)

	ResetExpectedErrors()
	expectedState.Store(expectedStateUnset)
	t.Setenv(
		ExpectedErrorsEnv,
		envValue,
	)

	t.Cleanup(
		func() {
			otel.SetTracerProvider(previous)
			ResetExpectedErrors()
			expectedState.Store(expectedStateUnset)
		},
	)

	buffer := &bytes.Buffer{}
	ctx := log.ContextWithLogger(
		t.Context(),
		log.New(
			log.WithWriter(buffer),
			log.WithLevel(log.LevelDebug),
		),
	)

	_, span := Start(
		ctx,
		"op",
	)
	buffer.Reset()

	return &expectedHarness{
		recorder: recorder,
		buffer:   buffer,
		span:     span,
	}
}

func (h *expectedHarness) endedSpan(t *testing.T) tracesdk.ReadOnlySpan {
	t.Helper()

	h.span.TraceSpan.End()

	ended := h.recorder.Ended()
	require.Len(
		t,
		ended,
		1,
	)

	return ended[0]
}

func TestErrExpectedMatchKeepsSpanOk(t *testing.T) {
	harness := newExpectedHarness(
		t,
		"",
	)
	RegisterExpectedErrors(isExpectedNotFound)

	err := harness.span.Err(errExpectedNotFound)
	require.Same(
		t,
		errExpectedNotFound,
		err,
	)

	ended := harness.endedSpan(t)
	require.Equal(
		t,
		codes.Unset,
		ended.Status().Code,
	)
	require.Empty(
		t,
		ended.Events(),
	)
	require.Contains(
		t,
		ended.Attributes(),
		attribute.Bool(
			"error.expected",
			true,
		),
	)

	out := harness.buffer.String()
	require.NotContains(
		t,
		out,
		`"level":"error"`,
	)
	require.Contains(
		t,
		out,
		`"level":"debug"`,
	)
	require.Contains(
		t,
		out,
		`"message":"Span recorded expected error"`,
	)
	require.Contains(
		t,
		out,
		`"error":"expected not found"`,
	)
}

func TestErrExpectedWrappedError(t *testing.T) {
	harness := newExpectedHarness(
		t,
		"",
	)
	RegisterExpectedErrors(isExpectedNotFound)

	wrapped := fmt.Errorf(
		"repo get: %w",
		errExpectedNotFound,
	)
	err := harness.span.Err(wrapped)
	require.Same(
		t,
		wrapped,
		err,
	)

	ended := harness.endedSpan(t)
	require.Equal(
		t,
		codes.Unset,
		ended.Status().Code,
	)
	require.NotContains(
		t,
		harness.buffer.String(),
		`"level":"error"`,
	)
}

func TestErrNotMatchingStaysError(t *testing.T) {
	harness := newExpectedHarness(
		t,
		"",
	)
	RegisterExpectedErrors(isExpectedNotFound)

	boom := errors.New("boom")
	err := harness.span.Err(boom)
	require.Same(
		t,
		boom,
		err,
	)

	ended := harness.endedSpan(t)
	require.Equal(
		t,
		codes.Error,
		ended.Status().Code,
	)
	require.Equal(
		t,
		"boom",
		ended.Status().Description,
	)
	require.Len(
		t,
		ended.Events(),
		1,
	)
	require.NotContains(
		t,
		ended.Attributes(),
		attribute.Bool(
			"error.expected",
			true,
		),
	)

	out := harness.buffer.String()
	require.Equal(
		t,
		1,
		strings.Count(
			out,
			`"level":"error"`,
		),
		out,
	)
	require.Contains(
		t,
		out,
		`"message":"Span recorded error"`,
	)
}

func TestErrEmptyRegistryStaysError(t *testing.T) {
	harness := newExpectedHarness(
		t,
		"",
	)

	require.Same(
		t,
		errExpectedNotFound,
		harness.span.Err(errExpectedNotFound),
	)

	ended := harness.endedSpan(t)
	require.Equal(
		t,
		codes.Error,
		ended.Status().Code,
	)
	require.Contains(
		t,
		harness.buffer.String(),
		`"level":"error"`,
	)
}

func TestErrSwitchOffStaysError(t *testing.T) {
	harness := newExpectedHarness(
		t,
		"off",
	)
	RegisterExpectedErrors(isExpectedNotFound)

	require.Same(
		t,
		errExpectedNotFound,
		harness.span.Err(errExpectedNotFound),
	)

	ended := harness.endedSpan(t)
	require.Equal(
		t,
		codes.Error,
		ended.Status().Code,
	)
	require.Contains(
		t,
		harness.buffer.String(),
		`"level":"error"`,
	)
}

func TestErrSwitchOffIsCaseInsensitive(t *testing.T) {
	harness := newExpectedHarness(
		t,
		" OFF ",
	)
	RegisterExpectedErrors(isExpectedNotFound)

	harness.span.Err(errExpectedNotFound)

	require.Equal(
		t,
		codes.Error,
		harness.endedSpan(t).Status().Code,
	)
}

func TestErrOtherEnvValueKeepsEnabled(t *testing.T) {
	harness := newExpectedHarness(
		t,
		"on",
	)
	RegisterExpectedErrors(isExpectedNotFound)

	harness.span.Err(errExpectedNotFound)

	require.Equal(
		t,
		codes.Unset,
		harness.endedSpan(t).Status().Code,
	)
}

func TestErrNilReturnsNilAndLogsNothing(t *testing.T) {
	harness := newExpectedHarness(
		t,
		"",
	)
	RegisterExpectedErrors(isExpectedNotFound)

	require.NoError(
		t,
		harness.span.Err(nil),
	)

	ended := harness.endedSpan(t)
	require.Equal(
		t,
		codes.Unset,
		ended.Status().Code,
	)
	require.Empty(
		t,
		ended.Attributes(),
	)
	require.Empty(
		t,
		harness.buffer.String(),
	)
}

func TestRegisterExpectedErrorsIgnoresNilAndAccumulates(t *testing.T) {
	newExpectedHarness(
		t,
		"",
	)

	RegisterExpectedErrors(nil)
	require.Nil(
		t,
		expectedPredicates.Load(),
	)

	RegisterExpectedErrors(
		isExpectedNotFound,
		nil,
	)
	RegisterExpectedErrors(
		func(err error) bool {
			return err.Error() == "second"
		},
	)

	require.True(
		t,
		isExpectedError(errExpectedNotFound),
	)
	require.True(
		t,
		isExpectedError(errors.New("second")),
	)
	require.False(
		t,
		isExpectedError(errors.New("third")),
	)
}

func TestResetExpectedErrors(t *testing.T) {
	newExpectedHarness(
		t,
		"",
	)
	RegisterExpectedErrors(isExpectedNotFound)
	require.True(
		t,
		isExpectedError(errExpectedNotFound),
	)

	ResetExpectedErrors()

	require.False(
		t,
		isExpectedError(errExpectedNotFound),
	)
}

func TestExpectedErrorsConcurrentRegisterAndRead(t *testing.T) {
	newExpectedHarness(
		t,
		"",
	)

	const workers = 8
	const rounds = 200

	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(2)

		go func() {
			defer group.Done()

			for round := 0; round < rounds; round++ {
				RegisterExpectedErrors(isExpectedNotFound)
			}
		}()

		go func() {
			defer group.Done()

			for round := 0; round < rounds; round++ {
				isExpectedError(errors.New("boom"))
				isExpectedError(errExpectedNotFound)
			}
		}()
	}
	group.Wait()

	require.Len(
		t,
		*expectedPredicates.Load(),
		workers*rounds,
	)
	require.True(
		t,
		isExpectedError(errExpectedNotFound),
	)
}
