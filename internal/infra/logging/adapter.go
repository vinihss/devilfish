package logging

import (
	"github.com/rs/zerolog"
)

// LoggerAdapter wraps zerolog.Logger to implement our Logger interface.
type LoggerAdapter struct {
	logger *zerolog.Logger
}

// NewLoggerAdapter creates a new LoggerAdapter.
func NewLoggerAdapter(logger *zerolog.Logger) *LoggerAdapter {
	return &LoggerAdapter{
		logger: logger,
	}
}

// Debug logs a debug message.
func (a *LoggerAdapter) Debug(msg string) {
	a.logger.Debug().Msg(msg)
}

// Debugf logs a debug message with formatting.
func (a *LoggerAdapter) Debugf(format string, args ...interface{}) {
	a.logger.Debug().Msgf(format, args...)
}

// Info logs an info message.
func (a *LoggerAdapter) Info(msg string) {
	a.logger.Info().Msg(msg)
}

// Infof logs an info message with formatting.
func (a *LoggerAdapter) Infof(format string, args ...interface{}) {
	a.logger.Info().Msgf(format, args...)
}

// Warn logs a warning message.
func (a *LoggerAdapter) Warn(msg string) {
	a.logger.Warn().Msg(msg)
}

// Warnf logs a warning message with formatting.
func (a *LoggerAdapter) Warnf(format string, args ...interface{}) {
	a.logger.Warn().Msgf(format, args...)
}

// Error logs an error message.
func (a *LoggerAdapter) Error(msg string) {
	a.logger.Error().Msg(msg)
}

// Errorf logs an error message with formatting.
func (a *LoggerAdapter) Errorf(format string, args ...interface{}) {
	a.logger.Error().Msgf(format, args...)
}

// With creates a sub-logger with the given fields.
func (a *LoggerAdapter) With(fields map[string]interface{}) Logger {
	subLogger := a.logger.With()
	for k, v := range fields {
		subLogger = subLogger.Interface(k, v)
	}

	newLogger := subLogger.Logger()
	return &LoggerAdapter{
		logger: &newLogger,
	}
}

// Ensure LoggerAdapter implements Logger
var _ Logger = (*LoggerAdapter)(nil)
