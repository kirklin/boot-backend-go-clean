package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

type slogLogger struct {
	logger *slog.Logger
	level  LogLevel
	opts   *slog.HandlerOptions
	output io.Writer
}

func NewSlogLogger(config *LoggerConfig) (Logger, error) {
	var output io.Writer = os.Stdout
	if config.Output != nil {
		output = config.Output
	}

	opts := &slog.HandlerOptions{
		Level:     convertToSlogLevel(config.Level),
		AddSource: config.EnableCaller,
	}

	var handler slog.Handler
	if config.Format == JSONFormat {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	if len(config.InitialFields) > 0 {
		attrs := make([]slog.Attr, 0, len(config.InitialFields))
		for k, v := range config.InitialFields {
			attrs = append(attrs, slog.Any(k, v))
		}
		handler = handler.WithAttrs(attrs)
	}

	logger := slog.New(handler)

	return &slogLogger{
		logger: logger,
		level:  config.Level,
		opts:   opts,
		output: output,
	}, nil
}

func (l *slogLogger) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

func (l *slogLogger) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}

func (l *slogLogger) Fatal(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
	os.Exit(1)
}

func (l *slogLogger) Panic(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
	panic(msg)
}

func (l *slogLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

func (l *slogLogger) Infof(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l *slogLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

func (l *slogLogger) Errorf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

func (l *slogLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

func (l *slogLogger) Panicf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.logger.Error(msg)
	panic(msg)
}

func (l *slogLogger) Log(context context.Context, level LogLevel, msg string, fields Fields) {
	l.logger.Log(context, convertToSlogLevel(level), msg, convertFields(fields)...)
}

func (l *slogLogger) LogWithDuration(level LogLevel, msg string, start time.Time, fields Fields) {
	duration := time.Since(start)
	newFields := Fields{"duration": duration.Milliseconds()} // 使用毫秒作为整数
	for k, v := range fields {
		newFields[k] = v
	}
	l.Log(context.Background(), level, msg, newFields)
}

func (l *slogLogger) ShouldLog(level LogLevel) bool {
	return level >= l.level
}

func (l *slogLogger) SetLevel(level LogLevel) {
	l.level = level
	l.opts.Level = convertToSlogLevel(level)

	// 重新创建 handler 以应用新的日志级别
	var handler slog.Handler
	if _, ok := l.logger.Handler().(*slog.JSONHandler); ok {
		handler = slog.NewJSONHandler(l.output, l.opts)
	} else {
		handler = slog.NewTextHandler(l.output, l.opts)
	}
	l.logger = slog.New(handler)
}

func (l *slogLogger) GetLevel() LogLevel {
	return l.level
}

func (l *slogLogger) SetFormat(format LogFormat) {
	var handler slog.Handler
	if format == JSONFormat {
		handler = slog.NewJSONHandler(l.output, l.opts)
	} else {
		handler = slog.NewTextHandler(l.output, l.opts)
	}
	l.logger = slog.New(handler)
}

func (l *slogLogger) Sync() error {
	// slog doesn't require explicit syncing
	return nil
}

func (l *slogLogger) Clone() Logger {
	return &slogLogger{
		logger: l.logger,
		level:  l.level,
		opts:   l.opts,
		output: l.output,
	}
}

func (l *slogLogger) SetOutput(w io.Writer) {
	l.output = w // 更新 output 字段
	var handler slog.Handler
	if _, ok := l.logger.Handler().(*slog.JSONHandler); ok {
		handler = slog.NewJSONHandler(l.output, l.opts)
	} else {
		handler = slog.NewTextHandler(l.output, l.opts)
	}
	l.logger = slog.New(handler)
}

func (l *slogLogger) With(keysAndValues ...interface{}) Logger {
	newLogger := l.logger.With(keysAndValues...)
	return &slogLogger{
		logger: newLogger,
		level:  l.level,
		opts:   l.opts,
		output: l.output,
	}
}

func (l *slogLogger) WithFields(fields Fields) Logger {
	newLogger := l.logger.With(convertFields(fields)...)
	return &slogLogger{
		logger: newLogger,
		level:  l.level,
		opts:   l.opts,
		output: l.output,
	}
}

func (l *slogLogger) WithName(name string) Logger {
	newLogger := l.logger.With("logger", name)
	return &slogLogger{
		logger: newLogger,
		level:  l.level,
		opts:   l.opts,
		output: l.output,
	}
}

func (l *slogLogger) WithTrace(traceID string) Logger {
	newLogger := l.logger.With("traceID", traceID)
	return &slogLogger{
		logger: newLogger,
		level:  l.level,
		opts:   l.opts,
		output: l.output,
	}
}

func (l *slogLogger) WithError(err error) Logger {
	newLogger := l.logger.With("error", err)
	return &slogLogger{
		logger: newLogger,
		level:  l.level,
		opts:   l.opts,
		output: l.output,
	}
}

func convertFields(fields Fields) []interface{} {
	result := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		result = append(result, k, v)
	}
	return result
}
