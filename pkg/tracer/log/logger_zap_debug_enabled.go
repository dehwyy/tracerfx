package log

import "go.uber.org/zap/zapcore"

func (z *zapLogger) DebugEnabled() bool {
	return z.core.Enabled(zapcore.DebugLevel)
}
