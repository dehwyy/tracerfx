package log

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func restoreGlobalLevel(t *testing.T) {
	t.Helper()

	previous := zerolog.GlobalLevel()
	t.Cleanup(
		func() {
			zerolog.SetGlobalLevel(previous)
		},
	)
}

func TestSetGlobalLevelFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want zerolog.Level
	}{
		{
			name: "debug",
			env:  "debug",
			want: zerolog.DebugLevel,
		},
		{
			name: "trace",
			env:  "trace",
			want: zerolog.TraceLevel,
		},
		{
			name: "info",
			env:  "info",
			want: zerolog.InfoLevel,
		},
		{
			name: "warn",
			env:  "warn",
			want: zerolog.WarnLevel,
		},
		{
			name: "error",
			env:  "error",
			want: zerolog.ErrorLevel,
		},
		{
			name: "upper case",
			env:  "DEBUG",
			want: zerolog.DebugLevel,
		},
		{
			name: "empty means info",
			env:  "",
			want: zerolog.InfoLevel,
		},
		{
			name: "invalid means info",
			env:  "loud",
			want: zerolog.InfoLevel,
		},
		{
			name: "spaces mean info",
			env:  "  ",
			want: zerolog.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				restoreGlobalLevel(t)
				zerolog.SetGlobalLevel(zerolog.PanicLevel)
				t.Setenv(
					LevelEnv,
					tt.env,
				)

				SetGlobalLevelFromEnv()

				require.Equal(
					t,
					tt.want,
					zerolog.GlobalLevel(),
				)
			},
		)
	}
}

func TestSetGlobalLevelFromEnvUnset(t *testing.T) {
	restoreGlobalLevel(t)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	SetGlobalLevelFromEnv()

	require.Equal(
		t,
		zerolog.InfoLevel,
		zerolog.GlobalLevel(),
	)
}

func TestSetGlobalLevelFromEnvRepeatedCalls(t *testing.T) {
	restoreGlobalLevel(t)
	t.Setenv(
		LevelEnv,
		"debug",
	)

	SetGlobalLevelFromEnv()
	SetGlobalLevelFromEnv()
	require.Equal(
		t,
		zerolog.DebugLevel,
		zerolog.GlobalLevel(),
	)

	t.Setenv(
		LevelEnv,
		"error",
	)
	SetGlobalLevelFromEnv()
	require.Equal(
		t,
		zerolog.ErrorLevel,
		zerolog.GlobalLevel(),
	)
}

func TestSetGlobalLevelFromEnvMatchesLoggerParsing(t *testing.T) {
	for _, value := range []string{"debug", "warn", "error", "info", "", "nope"} {
		t.Run(
			value,
			func(t *testing.T) {
				restoreGlobalLevel(t)
				t.Setenv(
					LevelEnv,
					value,
				)

				SetGlobalLevelFromEnv()

				require.Equal(
					t,
					parseEnvLevel(),
					zerolog.GlobalLevel(),
				)
			},
		)
	}
}

func TestGlobalLevelIsLowerBoundForLoggers(t *testing.T) {
	tests := []struct {
		name      string
		env       string
		wantDebug bool
	}{
		{
			name:      "info env hides debug",
			env:       "",
			wantDebug: false,
		},
		{
			name:      "debug env keeps debug",
			env:       "debug",
			wantDebug: true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				restoreGlobalLevel(t)
				t.Setenv(
					LevelEnv,
					tt.env,
				)
				SetGlobalLevelFromEnv()

				var buf bytes.Buffer
				logger := New(
					WithWriter(&buf),
					WithLevel(LevelDebug),
				)

				logger.Debug("noise")
				logger.Info("signal")

				out := buf.String()
				require.Contains(
					t,
					out,
					`"message":"signal"`,
				)
				require.Equal(
					t,
					tt.wantDebug,
					bytes.Contains(
						buf.Bytes(),
						[]byte(`"message":"noise"`),
					),
					out,
				)
			},
		)
	}
}
