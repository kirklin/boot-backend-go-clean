package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type zerologLogger struct {
	logger  zerolog.Logger
	level   LogLevel
	config  *LoggerConfig
	writers []io.Writer
}

func NewZerologLogger(config *LoggerConfig) (Logger, error) {
	var writers []io.Writer

	// 标准输出
	var output io.Writer = os.Stdout
	if config.Output != nil {
		output = config.Output
	}

	// 根据格式配置输出
	var consoleWriter io.Writer
	if config.Format == TextFormat {
		consoleWriter = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
		}
	} else {
		consoleWriter = output
	}
	writers = append(writers, consoleWriter)

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
	multiWriter := zerolog.MultiLevelWriter(writers...)

	// 设置日志级别
	level := convertToZerologLevel(config.Level)

	// 创建 zerolog.Logger
	zl := zerolog.New(multiWriter).Level(level).With().Timestamp().Logger()

	// 添加初始字段
	if len(config.InitialFields) > 0 {
		zl = zl.With().Fields(config.InitialFields).Logger()
	}

	// 启用调用者信息
	if config.EnableCaller {
		zl = zl.With().Caller().Logger()
	}

	// 启用堆栈跟踪
	if config.EnableStacktrace {
		zl = zl.With().Stack().Logger()
	}

	return &zerologLogger{
		logger:  zl,
		level:   config.Level,
		config:  config,
		writers: writers,
	}, nil
}

func (l *zerologLogger) Debug(msg string, args ...interface{}) {
	l.logger.Debug().Msgf(msg, args...)
}

func (l *zerologLogger) Info(msg string, args ...interface{}) {
	l.logger.Info().Msgf(msg, args...)
}

func (l *zerologLogger) Warn(msg string, args ...interface{}) {
	l.logger.Warn().Msgf(msg, args...)
}

func (l *zerologLogger) Error(msg string, args ...interface{}) {
	event := l.logger.Error()
	if l.config.EnableStacktrace {
		event = event.Stack()
	}
	event.Msgf(msg, args...)
}

func (l *zerologLogger) Fatal(msg string, args ...interface{}) {
	event := l.logger.Fatal()
	if l.config.EnableStacktrace {
		event = event.Stack()
	}
	event.Msgf(msg, args...)
}

func (l *zerologLogger) Panic(msg string, args ...interface{}) {
	l.logger.Panic().Msgf(msg, args...)
}

func (l *zerologLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}

func (l *zerologLogger) Infof(format string, args ...interface{}) {
	l.logger.Info().Msgf(format, args...)
}

func (l *zerologLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warn().Msgf(format, args...)
}

func (l *zerologLogger) Errorf(format string, args ...interface{}) {
	event := l.logger.Error()
	if l.config.EnableStacktrace {
		event = event.Stack()
	}
	event.Msgf(format, args...)
}

func (l *zerologLogger) Fatalf(format string, args ...interface{}) {
	event := l.logger.Fatal()
	if l.config.EnableStacktrace {
		event = event.Stack()
	}
	event.Msgf(format, args...)
}

func (l *zerologLogger) Panicf(format string, args ...interface{}) {
	l.logger.Panic().Msgf(format, args...)
}

func (l *zerologLogger) Log(ctx context.Context, level LogLevel, msg string, fields Fields) {
	event := l.logger.WithLevel(convertToZerologLevel(level))
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (l *zerologLogger) LogWithDuration(level LogLevel, msg string, start time.Time, fields Fields) {
	duration := time.Since(start)
	l.Log(context.Background(), level, msg, Fields{"duration": duration.Milliseconds()}.Merge(fields))
}

func (l *zerologLogger) ShouldLog(level LogLevel) bool {
	return l.level <= level
}

func (l *zerologLogger) SetLevel(level LogLevel) {
	l.level = level
	l.logger = l.logger.Level(convertToZerologLevel(level))
}

func (l *zerologLogger) GetLevel() LogLevel {
	return l.level
}

func (l *zerologLogger) Sync() error {
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

func (l *zerologLogger) Clone() Logger {
	return &zerologLogger{
		logger: l.logger,
		level:  l.level,
		config: l.config,
	}
}

func (l *zerologLogger) SetOutput(w io.Writer) {
	l.logger = l.logger.Output(w)
}

func (l *zerologLogger) With(keysAndValues ...interface{}) Logger {
	newLogger := l.logger.With().Fields(keysAndValues).Logger()
	return &zerologLogger{
		logger: newLogger,
		level:  l.level,
		config: l.config,
	}
}

func (l *zerologLogger) WithFields(fields Fields) Logger {
	newLogger := l.logger.With().Fields(map[string]interface{}(fields)).Logger()
	return &zerologLogger{
		logger: newLogger,
		level:  l.level,
		config: l.config,
	}
}

func (l *zerologLogger) WithName(name string) Logger {
	newLogger := l.logger.With().Str("logger", name).Logger()
	return &zerologLogger{
		logger: newLogger,
		level:  l.level,
		config: l.config,
	}
}

func (l *zerologLogger) WithTrace(traceID string) Logger {
	newLogger := l.logger.With().Str("traceID", traceID).Logger()
	return &zerologLogger{
		logger: newLogger,
		level:  l.level,
		config: l.config,
	}
}

func (l *zerologLogger) WithError(err error) Logger {
	newLogger := l.logger.With().Err(err).Logger()
	return &zerologLogger{
		logger: newLogger,
		level:  l.level,
		config: l.config,
	}
}

// Merge merges two Fields
func (f Fields) Merge(other Fields) Fields {
	for k, v := range other {
		f[k] = v
	}
	return f
}
