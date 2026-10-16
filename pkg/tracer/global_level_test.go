package tracer

import (
	"context"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestFxModuleSetsGlobalLevelAtBuild(t *testing.T) {
	restoreGlobalLevel(t)
	t.Cleanup(
		func() {
			log.SetDefaultLogger(nil)
		},
	)

	tests := []struct {
		name string
		env  string
		want zerolog.Level
	}{
		{
			name: "debug env",
			env:  "debug",
			want: zerolog.DebugLevel,
		},
		{
			name: "no env",
			env:  "",
			want: zerolog.InfoLevel,
		},
		{
			name: "invalid env",
			env:  "loud",
			want: zerolog.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				zerolog.SetGlobalLevel(zerolog.PanicLevel)
				t.Setenv(
					log.LevelEnv,
					tt.env,
				)

				FxModule()

				require.Equal(
					t,
					tt.want,
					zerolog.GlobalLevel(),
				)
			},
		)
	}
}

func TestProviderStartSetsGlobalLevel(t *testing.T) {
	restoreGlobalLevel(t)

	tests := []struct {
		name string
		env  string
		want zerolog.Level
	}{
		{
			name: "debug env",
			env:  "debug",
			want: zerolog.DebugLevel,
		},
		{
			name: "no env",
			env:  "",
			want: zerolog.InfoLevel,
		},
		{
			name: "invalid env",
			env:  "loud",
			want: zerolog.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				zerolog.SetGlobalLevel(zerolog.PanicLevel)
				t.Setenv(
					log.LevelEnv,
					tt.env,
				)

				provider := NewProvider(WithHost("127.0.0.1:1"))
				require.Equal(
					t,
					zerolog.PanicLevel,
					zerolog.GlobalLevel(),
				)

				require.NoError(
					t,
					provider.Start(t.Context()),
				)
				t.Cleanup(
					func() {
						require.NoError(
							t,
							provider.Stop(context.Background()),
						)
					},
				)

				require.Equal(
					t,
					tt.want,
					zerolog.GlobalLevel(),
				)
			},
		)
	}
}
