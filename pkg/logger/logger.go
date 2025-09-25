package logx

import (
	"github.com/Chative-core-poc-v1/server/internal/core"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var DefaultLoggerOpts = &LoggerOpts{
	Environment: core.Development,
}

type LoggerOpts struct {
	Environment core.Environment
}

func safe(otps ...LoggerOpts) *LoggerOpts {
	if len(otps) == 0 {
		return DefaultLoggerOpts
	}
	return &otps[0]
}

func Init(otps ...LoggerOpts) {
	if safe(otps...).Environment == core.Production {
		log.Logger = log.Logger.Level(zerolog.InfoLevel)
	} else {
		log.Logger = zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Caller().Logger()
		log.Logger = log.Logger.Level(zerolog.DebugLevel)
	}
}

func Debug() *zerolog.Event {
	return log.Debug()
}

func Info() *zerolog.Event {
	return log.Info()
}

func Warn() *zerolog.Event {
	return log.Warn()
}

func Error() *zerolog.Event {
	return log.Error()
}

func Panic() *zerolog.Event {
	return log.Panic()
}

func Fatal() *zerolog.Event {
	return log.Fatal()
}
