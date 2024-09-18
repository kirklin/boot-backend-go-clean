package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type zerologLogger struct {
	logger zerolog.Logger
	level  LogLevel
	config *LoggerConfig
}

func NewZerologLogger(config *LoggerConfig) (Logger, error) {
	// 设置输出
	var output io.Writer = os.Stdout
	if config.Output != nil {
		output = config.Output
	}

	// 设置日志级别
	level := convertToZerologLevel(config.Level)

	// 创建zerolog.Logger
	zl := zerolog.New(output).Level(level).With().Timestamp().Logger()

	// 设置日志格式
	switch config.Format {
	case JSONFormat:
		// JSON 是 zerolog 的默认格式，无需额外设置
	case TextFormat:
		zl = zl.Output(zerolog.ConsoleWriter{Out: output, TimeFormat: time.RFC3339})
	}

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
		logger: zl,
		level:  config.Level,
		config: config, // 保存配置以便后续使用
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
	// Zerolog doesn't require syncing
	return nil
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
