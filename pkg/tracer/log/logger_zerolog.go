package log

import (
	"io"
	"os"
	"sync"

	"github.com/rs/zerolog"
)

const LevelEnv = "TRACERFX_LOG_LEVEL"

var (
	levelOnce sync.Once
	level     zerolog.Level
)

func envLevel() zerolog.Level {
	levelOnce.Do(func() {
		level = zerolog.InfoLevel
		if v := os.Getenv(LevelEnv); v != "" {
			if parsed, err := zerolog.ParseLevel(v); err == nil {
				level = parsed
			}
		}
	})
	return level
}

type zerologLogger struct {
	logger zerolog.Logger
}

func NewZerologLogger() Logger {
	return newZerologLogger(os.Stdout, envLevel())
}

func newZerologLogger(w io.Writer, lvl zerolog.Level) Logger {
	logger := zerolog.New(w).Level(lvl).With().Timestamp().Logger()
	return &zerologLogger{logger: logger}
}

func (z *zerologLogger) Debug(msg string, args ...any) {
	event := z.logger.Debug()
	z.appendFields(event, args...)
	event.Msg(msg)
}

func (z *zerologLogger) Info(msg string, args ...any) {
	event := z.logger.Info()
	z.appendFields(event, args...)
	event.Msg(msg)
}

func (z *zerologLogger) Error(msg string, args ...any) {
	event := z.logger.Error()
	z.appendFields(event, args...)
	event.Msg(msg)
}

func (z *zerologLogger) With(args ...any) Logger {
	ctx := z.logger.With()
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				ctx = ctx.Interface(key, args[i+1])
			}
		}
	}
	return &zerologLogger{logger: ctx.Logger()}
}

func (z *zerologLogger) appendFields(event *zerolog.Event, args ...any) {
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				event.Interface(key, args[i+1])
			}
		}
	}
}
