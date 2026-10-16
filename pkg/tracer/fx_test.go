package tracer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestFxEventLoggerOption(t *testing.T) {
	restoreGlobalLevel(t)
	t.Cleanup(func() { log.SetDefaultLogger(nil) })

	var buf bytes.Buffer
	var provided log.Logger

	app := fx.New(
		FxModule(
			WithLogger(log.New(log.WithWriter(&buf))),
		),
		FxEventLogger(),
		fx.Populate(&provided),
	)

	require.NoError(t, app.Start(t.Context()))
	require.NoError(t, app.Stop(t.Context()))

	out := buf.String()
	require.Contains(t, out, `"message":"fx started"`)
	require.Contains(t, out, `"message":"fx hook OnStart executed"`)
	require.NotNil(t, provided)
	require.Same(t, provided, log.Default())
	require.False(t, strings.Contains(out, "[Fx]"))
}
