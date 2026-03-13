package log

import (
	"os"

	"github.com/rs/zerolog"
)

type zerologLogger struct {
	logger zerolog.Logger
}

func NewZerologLogger() Logger {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &zerologLogger{logger: logger}
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
