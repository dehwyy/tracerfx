package log

import "github.com/rs/zerolog"

func (z *zerologLogger) DebugEnabled() bool {
	return z.logger.GetLevel() <= zerolog.DebugLevel && zerolog.GlobalLevel() <= zerolog.DebugLevel
}
