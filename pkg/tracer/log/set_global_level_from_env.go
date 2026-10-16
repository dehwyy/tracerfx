package log

import (
	"github.com/rs/zerolog"
)

func SetGlobalLevelFromEnv() {
	zerolog.SetGlobalLevel(parseEnvLevel())
}
