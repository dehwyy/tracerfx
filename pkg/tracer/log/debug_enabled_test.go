package log

import (
	"bytes"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type plainLogger struct {
	mu    sync.Mutex
	withs int
}

func (p *plainLogger) Debug(string, ...any) {}
func (p *plainLogger) Info(string, ...any)  {}
func (p *plainLogger) Warn(string, ...any)  {}
func (p *plainLogger) Error(string, ...any) {}

func (p *plainLogger) With(...any) Logger {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.withs++

	return p
}

func (p *plainLogger) withCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.withs
}

type levelLogger struct {
	plainLogger
	enabled bool
}

func (l *levelLogger) DebugEnabled() bool {
	return l.enabled
}

func resetGlobalLevel(t *testing.T) {
	t.Helper()

	previous := zerolog.GlobalLevel()
	t.Cleanup(
		func() {
			zerolog.SetGlobalLevel(previous)
		},
	)
}

func TestDebugEnabledZerologLevels(t *testing.T) {
	resetGlobalLevel(t)
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	tests := []struct {
		name  string
		level Level
		want  bool
	}{
		{
			name:  "debug",
			level: LevelDebug,
			want:  true,
		},
		{
			name:  "info",
			level: LevelInfo,
			want:  false,
		},
		{
			name:  "warn",
			level: LevelWarn,
			want:  false,
		},
		{
			name:  "error",
			level: LevelError,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				logger := New(
					WithWriter(&bytes.Buffer{}),
					WithLevel(tt.level),
				)

				require.Equal(
					t,
					tt.want,
					DebugEnabled(logger),
				)
				require.Equal(
					t,
					tt.want,
					DebugEnabled(logger.With(
						"k",
						"v",
					)),
				)
			},
		)
	}
}

func TestDebugEnabledMatchesEmission(t *testing.T) {
	resetGlobalLevel(t)

	for _, global := range []zerolog.Level{zerolog.DebugLevel, zerolog.InfoLevel} {
		zerolog.SetGlobalLevel(global)

		var buf bytes.Buffer
		logger := New(
			WithWriter(&buf),
			WithLevel(LevelDebug),
		)
		logger.Debug("probe")

		require.Equal(
			t,
			buf.Len() > 0,
			DebugEnabled(logger),
			global.String(),
		)
	}
}

func TestDebugEnabledZerologGlobalLevel(t *testing.T) {
	resetGlobalLevel(t)

	logger := New(
		WithWriter(&bytes.Buffer{}),
		WithLevel(LevelDebug),
	)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	require.False(
		t,
		DebugEnabled(logger),
	)

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	require.True(
		t,
		DebugEnabled(logger),
	)
}

func TestDebugEnabledZap(t *testing.T) {
	tests := []struct {
		name  string
		level zapcore.Level
		want  bool
	}{
		{
			name:  "debug",
			level: zapcore.DebugLevel,
			want:  true,
		},
		{
			name:  "info",
			level: zapcore.InfoLevel,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				core := zapcore.NewCore(
					zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
					zapcore.AddSync(&bytes.Buffer{}),
					tt.level,
				)
				logger := NewZapLogger(zap.New(core))

				require.Equal(
					t,
					tt.want,
					DebugEnabled(logger),
				)
				require.Equal(
					t,
					tt.want,
					DebugEnabled(logger.With(
						"k",
						"v",
					)),
				)
			},
		)
	}
}

func TestDebugEnabledForeignLoggerIsEnabled(t *testing.T) {
	require.True(
		t,
		DebugEnabled(&plainLogger{}),
	)
}

func TestDebugEnabledForwardedBySpanLoggers(t *testing.T) {
	resetGlobalLevel(t)
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	sc := spanContext(
		t,
		true,
	)

	tests := []struct {
		name  string
		base  Logger
		level bool
	}{
		{
			name: "debug",
			base: New(
				WithWriter(&bytes.Buffer{}),
				WithLevel(LevelDebug),
			),
			level: true,
		},
		{
			name: "info",
			base: New(
				WithWriter(&bytes.Buffer{}),
				WithLevel(LevelInfo),
			),
			level: false,
		},
		{
			name: "foreign enabled",
			base: &levelLogger{
				enabled: true,
			},
			level: true,
		},
		{
			name: "foreign disabled",
			base: &levelLogger{
				enabled: false,
			},
			level: false,
		},
		{
			name:  "foreign plain",
			base:  &plainLogger{},
			level: true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name,
			func(t *testing.T) {
				require.Equal(
					t,
					tt.level,
					DebugEnabled(newSpanLogger(
						tt.base,
						sc,
					)),
				)
				require.Equal(
					t,
					tt.level,
					DebugEnabled(NewLazySpanLogger(
						tt.base,
						sc,
					)),
				)
			},
		)
	}
}

func TestDebugEnabledLazyDoesNotResolve(t *testing.T) {
	base := &levelLogger{
		enabled: false,
	}
	lazy := NewLazySpanLogger(
		base,
		spanContext(
			t,
			true,
		),
	)

	require.False(
		t,
		DebugEnabled(lazy),
	)
	require.Zero(
		t,
		base.withCalls(),
	)

	lazy.Info("resolve")
	require.Equal(
		t,
		1,
		base.withCalls(),
	)
}
