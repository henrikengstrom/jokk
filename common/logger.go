package common

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type ConsoleLogger struct {
	logger zerolog.Logger
}

type CacheLogger struct {
	cache []string
}

type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Panic(msg string)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Panicf(format string, args ...any)
}

func NewConsoleLogger() ConsoleLogger {
	return ConsoleLogger{
		logger: zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Logger(),
	}
}

func (l ConsoleLogger) Info(msg string) {
	l.logger.Info().Msg(msg)
}

func (l ConsoleLogger) Infof(msg string, args ...any) {
	l.logger.Info().Msg(fmt.Sprintf(msg, args...))
}

func (l ConsoleLogger) Warn(msg string) {
	l.logger.Warn().Msg(msg)
}

func (l ConsoleLogger) Warnf(msg string, args ...any) {
	l.logger.Warn().Msg(fmt.Sprintf(msg, args...))
}

func (l ConsoleLogger) Error(msg string) {
	l.logger.Error().Msg(msg)
}

func (l ConsoleLogger) Errorf(msg string, args ...any) {
	l.logger.Error().Msg(fmt.Sprintf(msg, args...))
}

func (l ConsoleLogger) Panic(msg string) {
	l.logger.Panic().Msg(msg)
}

func (l ConsoleLogger) Panicf(msg string, args ...any) {
	l.logger.Panic().Msg(fmt.Sprintf(msg, args...))
}

func NewCacheLogger() CacheLogger {
	return CacheLogger{
		cache: []string{},
	}
}

func (l *CacheLogger) Info(msg string) {
	l.cache = append(l.cache, msg)
}

func (l *CacheLogger) Infof(msg string, args ...any) {
	l.cache = append(l.cache, fmt.Sprintf(msg, args...))
}

func (l *CacheLogger) Warn(msg string) {
	l.cache = append(l.cache, msg)
}

func (l *CacheLogger) Warnf(msg string, args ...any) {
	l.cache = append(l.cache, fmt.Sprintf(msg, args...))
}

func (l *CacheLogger) Error(msg string) {
	l.cache = append(l.cache, msg)
}

func (l *CacheLogger) Errorf(msg string, args ...any) {
	l.cache = append(l.cache, fmt.Sprintf(msg, args...))
}

func (l *CacheLogger) Panic(msg string) {
	l.cache = append(l.cache, msg)
}

func (l *CacheLogger) Panicf(msg string, args ...any) {
	l.cache = append(l.cache, fmt.Sprintf(msg, args...))
}

func (l *CacheLogger) Content() string {
	var content string
	for _, s := range l.cache {
		content = content + s
	}
	return content
}

func (l *CacheLogger) Clear() {
	l.cache = []string{}
}
