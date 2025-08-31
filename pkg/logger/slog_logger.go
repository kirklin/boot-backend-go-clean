package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type slogLogger struct {
	logger  *slog.Logger
	level   LogLevel
	opts    *slog.HandlerOptions
	output  io.Writer
	writers []io.Writer
}

func NewSlogLogger(config *LoggerConfig) (Logger, error) {
	var writers []io.Writer

	// 标准输出
	var output io.Writer = os.Stdout
	if config.Output != nil {
		output = config.Output
	}
	writers = append(writers, output)

	// 文件输出
	if config.FileConfig != nil && config.FileConfig.Enable {
		// 构建完整的日志路径
		logDir := filepath.Join(config.FileConfig.Directory, config.FileConfig.Environment)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("创建日志目录失败: %v", err)
		}

		filename := filepath.Join(logDir, config.FileConfig.Filename)

		fileWriter := &lumberjack.Logger{
			Filename:   filename,
			MaxSize:    config.FileConfig.MaxSize,
			MaxBackups: config.FileConfig.MaxBackups,
			MaxAge:     config.FileConfig.MaxAge,
			Compress:   config.FileConfig.Compress,
			LocalTime:  config.FileConfig.UseLocalTime,
		}
		writers = append(writers, fileWriter)
	}

	// 使用 MultiWriter 组合所有输出
	multiWriter := io.MultiWriter(writers...)

	opts := &slog.HandlerOptions{
		Level:     convertToSlogLevel(config.Level),
		AddSource: config.EnableCaller,
	}

	var handler slog.Handler
	if config.Format == JSONFormat {
		handler = slog.NewJSONHandler(multiWriter, opts)
	} else {
		handler = slog.NewTextHandler(multiWriter, opts)
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
		logger:  logger,
		level:   config.Level,
		opts:    opts,
		output:  multiWriter,
		writers: writers,
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
	var syncErr error
	for _, writer := range l.writers {
		if syncer, ok := writer.(interface{ Sync() error }); ok {
			if err := syncer.Sync(); err != nil {
				syncErr = err
			}
		}
	}
	return syncErr
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
