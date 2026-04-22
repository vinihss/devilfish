package logging

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger interface defines the logging operations required by providers.
// This allows different logger implementations to be used.
type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...interface{})
	Info(msg string)
	Infof(format string, args ...interface{})
	Warn(msg string)
	Warnf(format string, args ...interface{})
	Error(msg string)
	Errorf(format string, args ...interface{})
	With(fields map[string]interface{}) Logger
}

// ZLogger wraps zerolog Logger with our own interface.
type ZLogger struct {
	logger zerolog.Logger
	mu     sync.RWMutex
}

// Level represents the logging level.
type Level string

const (
	// DebugLevel is the debug log level.
	DebugLevel Level = "debug"
	// InfoLevel is the info log level.
	InfoLevel Level = "info"
	// WarnLevel is the warn log level.
	WarnLevel Level = "warn"
	// ErrorLevel is the error log level.
	ErrorLevel Level = "error"
)

// Format represents the log format.
type Format string

const (
	// JSONFormat is the JSON log format.
	JSONFormat Format = "json"
	// TextFormat is the text log format.
	TextFormat Format = "text"
)

// DefaultLogger is the global logger instance.
var DefaultLogger *ZLogger

// LevelMapping maps our Level to zerolog Level.
var LevelMapping = map[Level]zerolog.Level{
	DebugLevel: zerolog.DebugLevel,
	InfoLevel:  zerolog.InfoLevel,
	WarnLevel:  zerolog.WarnLevel,
	ErrorLevel: zerolog.ErrorLevel,
}

// FormatMapping maps our Format to zerolog Format.
var FormatMapping = map[Format]func() io.Writer{
	JSONFormat: func() io.Writer { return os.Stdout },
	TextFormat: func() io.Writer { return os.Stdout },
}

// Option is a functional option for configuring the logger.
type Option func(*ZLogger)

// WithLevel sets the log level.
func WithLevel(level Level) Option {
	return func(l *ZLogger) {
		zerolog.SetGlobalLevel(LevelMapping[level])
	}
}

// WithOutput sets the log output.
func WithOutput(output string) Option {
	return func(l *ZLogger) {
		var w io.Writer
		switch output {
		case "stdout":
			w = os.Stdout
		case "stderr":
			w = os.Stderr
		default:
			// Assume it's a file path
			f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
				w = os.Stdout
			} else {
				w = f
			}
		}
		l.logger = l.logger.Output(w)
	}
}

// WithFormat sets the log format.
func WithFormat(format Format) Option {
	return func(l *ZLogger) {
		if format == TextFormat {
			l.logger = l.logger.Output(zerolog.ConsoleWriter{
				TimeFormat: time.RFC3339,
			})
		}
	}
}

// WithTimestamp enables timestamp in logs.
func WithTimestamp() Option {
	return func(l *ZLogger) {
		l.logger = l.logger.With().Timestamp().Logger()
	}
}

// WithCaller enables caller in logs.
func WithCaller() Option {
	return func(l *ZLogger) {
		l.logger = l.logger.With().Caller().Logger()
	}
}

// New creates a new ZLogger with the given options.
func New(opts ...Option) *ZLogger {
	l := &ZLogger{
		logger: log.Logger,
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// NewLogger creates a new ZLogger. It accepts a config interface to avoid circular imports.
// If cfg is nil, default settings are used.
func NewLogger(cfg interface{}) *ZLogger {
	// Use defaults if no config provided
	return New(
		WithLevel(InfoLevel),
		WithOutput("stdout"),
		WithFormat(JSONFormat),
		WithTimestamp(),
	)
}

// Init initializes the DefaultLogger with the given options.
func Init(opts ...Option) {
	DefaultLogger = New(opts...)
}

// MustInit initializes the DefaultLogger and panics on error.
func MustInit(level Level, format Format, output string) {
	l := New(
		WithLevel(level),
		WithOutput(output),
		WithFormat(format),
		WithTimestamp(),
	)
	DefaultLogger = l
}

// Log methods

// Debug logs a debug message.
func (l *ZLogger) Debug(msg string) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Debug().Msg(msg)
}

// Debugf logs a debug message with formatting.
func (l *ZLogger) Debugf(format string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Debug().Msgf(format, args...)
}

// Info logs an info message.
func (l *ZLogger) Info(msg string) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Info().Msg(msg)
}

// Infof logs an info message with formatting.
func (l *ZLogger) Infof(format string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Info().Msgf(format, args...)
}

// Warn logs a warning message.
func (l *ZLogger) Warn(msg string) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Warn().Msg(msg)
}

// Warnf logs a warning message with formatting.
func (l *ZLogger) Warnf(format string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Warn().Msgf(format, args...)
}

// Error logs an error message.
func (l *ZLogger) Error(msg string) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Error().Msg(msg)
}

// Errorf logs an error message with formatting.
func (l *ZLogger) Errorf(format string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	l.logger.Error().Msgf(format, args...)
}

// With creates a sub-logger with the given fields.
func (l *ZLogger) With(fields map[string]interface{}) *ZLogger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	subLogger := l.logger.With()
	for k, v := range fields {
		subLogger = subLogger.Interface(k, v)
	}

	return &ZLogger{
		logger: subLogger.Logger(),
	}
}

// Named creates a sub-logger with the given name.
func (l *ZLogger) Named(name string) *ZLogger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return &ZLogger{
		logger: l.logger.With().Str("name", name).Logger(),
	}
}

// Sync flushes any buffered log entries.
func (l *ZLogger) Sync() {
	// zerolog doesn't require explicit sync
	// but we can call sync on the output file if needed
}

// Package-level convenience methods using DefaultLogger

// Debug logs a debug message using the default logger.
func Debug(msg string) {
	if DefaultLogger != nil {
		DefaultLogger.Debug(msg)
	}
}

// Debugf logs a debug message with formatting using the default logger.
func Debugf(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Debugf(format, args...)
	}
}

// Info logs an info message using the default logger.
func Info(msg string) {
	if DefaultLogger != nil {
		DefaultLogger.Info(msg)
	}
}

// Infof logs an info message with formatting using the default logger.
func Infof(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Infof(format, args...)
	}
}

// Warn logs a warning message using the default logger.
func Warn(msg string) {
	if DefaultLogger != nil {
		DefaultLogger.Warn(msg)
	}
}

// Warnf logs a warning message with formatting using the default logger.
func Warnf(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Warnf(format, args...)
	}
}

// Error logs an error message using the default logger.
func Error(msg string) {
	if DefaultLogger != nil {
		DefaultLogger.Error(msg)
	}
}

// Errorf logs an error message with formatting using the default logger.
func Errorf(format string, args ...interface{}) {
	if DefaultLogger != nil {
		DefaultLogger.Errorf(format, args...)
	}
}

// With creates a sub-logger with the given fields using the default logger.
func With(fields map[string]interface{}) *ZLogger {
	if DefaultLogger != nil {
		return DefaultLogger.With(fields)
	}
	return New()
}

// Named creates a sub-logger with the given name using the default logger.
func Named(name string) *ZLogger {
	if DefaultLogger != nil {
		return DefaultLogger.Named(name)
	}
	return New()
}
