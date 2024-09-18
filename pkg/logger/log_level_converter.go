package logger

import (
	"github.com/rs/zerolog"
	"go.uber.org/zap/zapcore"
	"log/slog"
)

// LogLevelConverter handles conversion between different logging libraries.
type LogLevelConverter struct{}

// convertToZapLevel converts LogLevel to zapcore.Level
func convertToZapLevel(level LogLevel) zapcore.Level {
	switch level {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	case FatalLevel:
		return zapcore.FatalLevel
	case PanicLevel:
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}

// convertToSlogLevel converts LogLevel to slog.Level
func convertToSlogLevel(level LogLevel) slog.Level {
	switch level {
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	case FatalLevel, PanicLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// convertToZerologLevel converts LogLevel to zerolog.Level
func convertToZerologLevel(level LogLevel) zerolog.Level {
	switch level {
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case FatalLevel:
		return zerolog.FatalLevel
	case PanicLevel:
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}
