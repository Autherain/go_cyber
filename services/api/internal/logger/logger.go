package logger

import (
	"fmt"
	"log/slog"
	"os"
	"time"
)

type (
	Format   string
	LogLevel string
)

const (
	JSONFormat Format = "json"
	TextFormat Format = "text"

	TraceLevel LogLevel = "trace"
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
)

func (f *Format) UnmarshalText(text []byte) error {
	switch string(text) {
	case string(JSONFormat), string(TextFormat):
		*f = Format(text)
		return nil
	default:
		*f = TextFormat
		return nil
	}
}

func (l *LogLevel) UnmarshalText(text []byte) error {
	switch string(text) {
	case string(TraceLevel), string(DebugLevel), string(InfoLevel), string(WarnLevel), string(ErrorLevel):
		*l = LogLevel(text)
		return nil
	default:
		*l = InfoLevel
		return nil
	}
}

type Config struct {
	Format    Format
	Level     LogLevel
	AddSource bool
}

// Logger is the main logger struct that handles both standard and RES logging
type Logger struct {
	slog *slog.Logger
}

// NewLogger creates a new configured logger that can be used for both standard and RES logging
func NewLogger(cfg Config) *Logger {
	level := convertLevel(cfg.Level)
	handler := createHandler(cfg, level)
	return &Logger{slog: slog.New(handler)}
}

func NewDefault() *Logger {
	defaultConfig := Config{
		Format:    TextFormat,
		Level:     InfoLevel,
		AddSource: false,
	}
	return NewLogger(defaultConfig)
}

func convertLevel(level LogLevel) slog.Level {
	switch level {
	case TraceLevel, DebugLevel:
		return slog.LevelDebug
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func createHandler(cfg Config, level slog.Level) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   slog.TimeKey,
					Value: slog.StringValue(time.Now().Format(time.RFC3339)),
				}
			}
			return a
		},
	}

	if cfg.Format == JSONFormat {
		return slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.NewTextHandler(os.Stdout, opts)
}

// Standard logging methods
func (l *Logger) Trace(msg string, args ...any) { l.slog.Debug(msg, args...) }
func (l *Logger) Debug(msg string, args ...any) { l.slog.Debug(msg, args...) }
func (l *Logger) Info(msg string, args ...any)  { l.slog.Info(msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.slog.Warn(msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.slog.Error(msg, args...) }

// RES-compatible logging methods
func (l *Logger) Tracef(format string, v ...interface{}) { l.Trace(fmt.Sprintf(format, v...)) }
func (l *Logger) Debugf(format string, v ...interface{}) { l.Debug(fmt.Sprintf(format, v...)) }
func (l *Logger) Infof(format string, v ...interface{})  { l.Info(fmt.Sprintf(format, v...)) }
func (l *Logger) Warnf(format string, v ...interface{})  { l.Warn(fmt.Sprintf(format, v...)) }
func (l *Logger) Errorf(format string, v ...interface{}) { l.Error(fmt.Sprintf(format, v...)) }

// SlogLogger returns the underlying slog.Logger if needed
func (l *Logger) SlogLogger() *slog.Logger {
	return l.slog
}
