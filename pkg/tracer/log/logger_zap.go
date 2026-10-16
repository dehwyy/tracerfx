package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	logger *zap.SugaredLogger
	core   zapcore.Core
}

func NewZapLogger(logger *zap.Logger) Logger {
	if logger == nil {
		production, err := zap.NewProduction()
		if err != nil {
			production = zap.NewNop()
		}
		logger = production
	}
	return &zapLogger{
		logger: logger.Sugar(),
		core:   logger.Core(),
	}
}

func (z *zapLogger) Debug(msg string, args ...any) {
	z.logger.Debugw(msg, normalizeFields(args)...)
}

func (z *zapLogger) Info(msg string, args ...any) {
	z.logger.Infow(msg, normalizeFields(args)...)
}

func (z *zapLogger) Warn(msg string, args ...any) {
	z.logger.Warnw(msg, normalizeFields(args)...)
}

func (z *zapLogger) Error(msg string, args ...any) {
	z.logger.Errorw(msg, normalizeFields(args)...)
}

func (z *zapLogger) With(args ...any) Logger {
	return &zapLogger{
		logger: z.logger.With(normalizeFields(args)...),
		core:   z.core,
	}
}
