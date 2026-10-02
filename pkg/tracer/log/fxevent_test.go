package log

import (
	"bytes"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxevent"
)

func TestFxEventLogger(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name      string
		event     fxevent.Event
		wantLevel string
		wantMsg   string
		wantErr   string
	}{
		{name: "started", event: &fxevent.Started{}, wantLevel: "info", wantMsg: "fx started"},
		{name: "start failed", event: &fxevent.Started{Err: boom}, wantLevel: "error", wantMsg: "fx start failed", wantErr: "boom"},
		{name: "on start executed", event: &fxevent.OnStartExecuted{FunctionName: "f", Runtime: time.Second}, wantLevel: "info", wantMsg: "fx hook OnStart executed"},
		{name: "on start failed", event: &fxevent.OnStartExecuted{FunctionName: "f", Err: boom}, wantLevel: "error", wantMsg: "fx hook OnStart failed", wantErr: "boom"},
		{name: "on stop failed", event: &fxevent.OnStopExecuted{FunctionName: "f", Err: boom}, wantLevel: "error", wantMsg: "fx hook OnStop failed", wantErr: "boom"},
		{name: "provided", event: &fxevent.Provided{ConstructorName: "c", OutputTypeNames: []string{"a", "b"}}, wantLevel: "debug", wantMsg: "fx provided"},
		{name: "provide failed", event: &fxevent.Provided{ConstructorName: "c", Err: boom}, wantLevel: "error", wantMsg: "fx provide failed", wantErr: "boom"},
		{name: "invoke failed", event: &fxevent.Invoked{FunctionName: "f", Err: boom}, wantLevel: "error", wantMsg: "fx invoke failed", wantErr: "boom"},
		{name: "rolling back", event: &fxevent.RollingBack{StartErr: boom}, wantLevel: "error", wantMsg: "fx start failed, rolling back", wantErr: "boom"},
		{name: "stopping", event: &fxevent.Stopping{Signal: os.Signal(syscall.SIGTERM)}, wantLevel: "info", wantMsg: "fx stopping"},
		{name: "stopped", event: &fxevent.Stopped{}, wantLevel: "info", wantMsg: "fx stopped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := NewFxEventLogger(New(WithWriter(&buf), WithLevel(LevelDebug)))

			l.LogEvent(tt.event)

			entries := decode(t, &buf)
			require.Len(t, entries, 1)
			require.Equal(t, tt.wantLevel, entries[0]["level"])
			require.Equal(t, tt.wantMsg, entries[0]["message"])
			if tt.wantErr != "" {
				require.Equal(t, tt.wantErr, entries[0]["error"])
			}
		})
	}
}

func TestFxEventLoggerSilentOnSuccessfulInvoke(t *testing.T) {
	var buf bytes.Buffer
	NewFxEventLogger(New(WithWriter(&buf), WithLevel(LevelDebug))).LogEvent(&fxevent.Invoked{FunctionName: "f"})
	require.Empty(t, buf.String())
}
