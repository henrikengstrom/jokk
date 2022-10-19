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

type DevNullLogger struct {
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
	ContentString() string
	Content() []string
	Clear()
}

// DEV NULL LOGGER

func NewDevNullLogger() DevNullLogger {
	return DevNullLogger{}
}

func (l DevNullLogger) Info(msg string)                   {}
func (l DevNullLogger) Warn(msg string)                   {}
func (l DevNullLogger) Error(msg string)                  {}
func (l DevNullLogger) Panic(msg string)                  {}
func (l DevNullLogger) Infof(format string, args ...any)  {}
func (l DevNullLogger) Warnf(format string, args ...any)  {}
func (l DevNullLogger) Errorf(format string, args ...any) {}
func (l DevNullLogger) Panicf(format string, args ...any) {}
func (l DevNullLogger) ContentString() string {
	return ""
}
func (l DevNullLogger) Content() []string {
	return []string{""}
}
func (l DevNullLogger) Clear() {}

// CONSOLE LOGGER

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

func (l ConsoleLogger) ContentString() string {
	return ""
}

func (l ConsoleLogger) Content() []string {
	return []string{""}
}

func (l ConsoleLogger) Clear() {}

// CACHE LOGGER

func NewCacheLogger() *CacheLogger {
	return &CacheLogger{
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

func (l *CacheLogger) ContentString() string {
	var content string
	for _, s := range l.cache {
		content = content + s
	}
	return content
}

func (l *CacheLogger) Content() []string {
	return l.cache
}

func (l *CacheLogger) Clear() {
	l.cache = []string{}
}
