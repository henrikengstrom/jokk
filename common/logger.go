package common

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type JokkLogger struct {
	logger zerolog.Logger
}

type Logger interface {
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Panic(args ...any)

	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Panicf(format string, args ...any)
}

func NewLogger() JokkLogger {
	return JokkLogger{
		logger: zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Logger(),
	}
}

func (jl JokkLogger) Info(m string) {
	jl.logger.Info().Msg(m)
}

func (jl JokkLogger) Infof(m string, args ...any) {
	jl.logger.Info().Msg(fmt.Sprintf(m, args...))
}

func (jl JokkLogger) Warn(m string) {
	jl.logger.Warn().Msg(m)
}

func (jl JokkLogger) Warnf(m string, args ...any) {
	jl.logger.Warn().Msg(fmt.Sprintf(m, args...))
}

func (jl JokkLogger) Error(m string) {
	jl.logger.Error().Msg(m)
}

func (jl JokkLogger) Errorf(m string, args ...any) {
	jl.logger.Error().Msg(fmt.Sprintf(m, args...))
}

func (jl JokkLogger) Panic(m string) {
	jl.logger.Panic().Msg(m)
}

func (jl JokkLogger) Panicf(m string, args ...any) {
	jl.logger.Panic().Msg(fmt.Sprintf(m, args...))
}
