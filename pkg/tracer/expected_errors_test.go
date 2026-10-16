package tracer

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/dspan"
	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

var errTestExpected = errors.New("test expected")

func restoreGlobalLevel(t *testing.T) {
	t.Helper()

	previous := zerolog.GlobalLevel()
	t.Cleanup(
		func() {
			zerolog.SetGlobalLevel(previous)
		},
	)
}

func resetExpectedErrors(t *testing.T) {
	t.Helper()

	dspan.ResetExpectedErrors()
	t.Cleanup(dspan.ResetExpectedErrors)
}

func countingPredicate(calls *int, expected error) func(error) bool {
	return func(err error) bool {
		*calls++

		return errors.Is(
			err,
			expected,
		)
	}
}

func TestNewProviderRegistersExpectedErrorsOnce(t *testing.T) {
	resetExpectedErrors(t)

	calls := 0
	NewProvider(
		WithExpectedErrors(
			countingPredicate(
				&calls,
				errTestExpected,
			),
		),
	)
	require.Zero(
		t,
		calls,
	)

	buffer := &bytes.Buffer{}
	ctx := log.ContextWithLogger(
		t.Context(),
		log.New(
			log.WithWriter(buffer),
			log.WithLevel(log.LevelDebug),
		),
	)

	_, span := dspan.Start(
		ctx,
		"op",
	)
	buffer.Reset()

	other := errors.New("other")
	require.Same(
		t,
		other,
		span.Err(other),
	)
	require.Equal(
		t,
		1,
		calls,
	)
	require.Contains(
		t,
		buffer.String(),
		`"level":"error"`,
	)

	buffer.Reset()
	require.Same(
		t,
		errTestExpected,
		span.Err(errTestExpected),
	)
	require.Equal(
		t,
		2,
		calls,
	)
	require.NotContains(
		t,
		buffer.String(),
		`"level":"error"`,
	)
}

func TestFxModuleRegistersExpectedErrorsAtBuildOnce(t *testing.T) {
	resetExpectedErrors(t)
	restoreGlobalLevel(t)
	t.Setenv(
		log.LevelEnv,
		"debug",
	)

	calls := 0
	buffer := &bytes.Buffer{}
	module := FxModule(
		WithLogger(
			log.New(
				log.WithWriter(buffer),
				log.WithLevel(log.LevelDebug),
			),
		),
		WithExpectedErrors(
			countingPredicate(
				&calls,
				errTestExpected,
			),
		),
	)
	t.Cleanup(
		func() {
			log.SetDefaultLogger(nil)
		},
	)

	_, before := dspan.Start(
		t.Context(),
		"before",
	)
	buffer.Reset()
	require.Same(
		t,
		errTestExpected,
		before.Err(errTestExpected),
	)
	require.Equal(
		t,
		1,
		calls,
	)

	app := fx.New(
		module,
		fx.NopLogger,
	)
	require.NoError(
		t,
		app.Start(t.Context()),
	)
	t.Cleanup(
		func() {
			require.NoError(
				t,
				app.Stop(context.Background()),
			)
		},
	)

	calls = 0
	buffer.Reset()

	_, span := dspan.Start(
		t.Context(),
		"after",
	)
	buffer.Reset()

	other := errors.New("other")
	require.Same(
		t,
		other,
		span.Err(other),
	)
	require.Equal(
		t,
		1,
		calls,
	)
	require.Contains(
		t,
		buffer.String(),
		`"level":"error"`,
	)

	buffer.Reset()
	require.Same(
		t,
		errTestExpected,
		span.Err(errTestExpected),
	)
	require.Contains(
		t,
		buffer.String(),
		`"message":"Span recorded expected error"`,
	)
	require.NotContains(
		t,
		buffer.String(),
		`"level":"error"`,
	)
}

func TestWithExpectedErrorsAccumulates(t *testing.T) {
	first := func(err error) bool { return false }
	second := func(err error) bool { return true }

	opts := newOptions(
		WithExpectedErrors(first),
		WithExpectedErrors(second),
	)

	require.Len(
		t,
		opts.expectedErrors,
		2,
	)
	require.False(
		t,
		opts.expectedRegistered,
	)
}
