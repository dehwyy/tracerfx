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

func parseEnvLevel() zerolog.Level {
	if v := os.Getenv(LevelEnv); v != "" {
		if parsed, err := zerolog.ParseLevel(v); err == nil {
			return parsed
		}
	}
	return zerolog.InfoLevel
}

func envLevel() zerolog.Level {
	levelOnce.Do(func() {
		level = parseEnvLevel()
	})
	return level
}

type Option func(*config)

type config struct {
	writer io.Writer
	level  zerolog.Level
}

func WithWriter(w io.Writer) Option {
	return func(c *config) {
		c.writer = w
	}
}

func WithLevel(l Level) Option {
	return func(c *config) {
		c.level = l.zerolog()
	}
}

func New(opts ...Option) Logger {
	cfg := config{
		writer: os.Stdout,
		level:  envLevel(),
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return newZerologLogger(cfg.writer, cfg.level)
}

type zerologLogger struct {
	logger zerolog.Logger
}

func NewZerologLogger() Logger {
	return New()
}

func newZerologLogger(w io.Writer, lvl zerolog.Level) Logger {
	logger := zerolog.New(w).Level(lvl).With().Timestamp().Logger()
	return &zerologLogger{logger: logger}
}

func (z *zerologLogger) Debug(msg string, args ...any) {
	z.emit(z.logger.Debug(), msg, args)
}

func (z *zerologLogger) Info(msg string, args ...any) {
	z.emit(z.logger.Info(), msg, args)
}

func (z *zerologLogger) Warn(msg string, args ...any) {
	z.emit(z.logger.Warn(), msg, args)
}

func (z *zerologLogger) Error(msg string, args ...any) {
	z.emit(z.logger.Error(), msg, args)
}

func (z *zerologLogger) With(args ...any) Logger {
	if len(args) == 0 {
		return z
	}

	child := z.logger.With().Fields(normalizeFields(args)).Logger()

	return &zerologLogger{logger: child}
}

func (z *zerologLogger) emit(event *zerolog.Event, msg string, args []any) {
	if event == nil {
		return
	}
	if len(args) > 0 {
		event = event.Fields(normalizeFields(args))
	}
	event.Msg(msg)
}
