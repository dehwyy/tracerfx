package log

import "go.uber.org/zap"

type zapLogger struct {
	logger *zap.SugaredLogger
}

func NewZapLogger(logger *zap.Logger) Logger {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &zapLogger{logger: logger.Sugar()}
}

func (z *zapLogger) Info(msg string, args ...any) {
	z.logger.Infow(msg, args...)
}

func (z *zapLogger) Error(msg string, args ...any) {
	z.logger.Errorw(msg, args...)
}

func (z *zapLogger) With(args ...any) Logger {
	return &zapLogger{
		logger: z.logger.With(args...),
	}
}
