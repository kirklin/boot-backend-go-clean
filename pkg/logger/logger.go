package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"
)

// Global logger instance
var globalLogger Logger

// InitLogger initializes the global logger instance
func InitLogger(config *LoggerConfig) error {
	logger, err := NewLogger(config, "zap")
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	globalLogger = logger
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() Logger {
	if globalLogger == nil {
		// If global logger is not initialized, initialize with default configuration
		config := NewDefaultConfig()
		if err := InitLogger(config); err != nil {
			// At this point, the logger is not initialized, so we use the standard log package.
			panic(fmt.Sprintf("Failed to initialize default logger: %v", err))
		}
	}
	return globalLogger
}

// LogLevel defines the severity of a log message
// LogLevel 定义日志消息的严重程度
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	PanicLevel
)

// LogFormat defines the output format of log messages
// LogFormat 定义日志消息的输出格式
type LogFormat int

const (
	JSONFormat LogFormat = iota
	TextFormat
)

// Fields defines a map of field names to values
// Fields 定义字段名到值的映射
type Fields map[string]interface{}

// Logger defines the interface for logging operations
// Logger 定义日志操作的接口
type Logger interface {

	// Debug logs a message at DebugLevel
	// Debug 在DebugLevel级别记录消息
	Debug(msg string, args ...interface{})

	// Info logs a message at InfoLevel
	// Info 在InfoLevel级别记录消息
	Info(msg string, args ...interface{})

	// Warn logs a message at WarnLevel
	// Warn 在WarnLevel级别记录消息
	Warn(msg string, args ...interface{})

	// Error logs a message at ErrorLevel
	// Error 在ErrorLevel级别记录消息
	Error(msg string, args ...interface{})

	// Fatal logs a message at FatalLevel and then calls os.Exit(1)
	// Fatal 在FatalLevel级别记录消息，然后调用os.Exit(1)
	Fatal(msg string, args ...interface{})

	// Panic logs a message at PanicLevel and then panics
	// Panic 在PanicLevel级别记录消息，然后触发panic
	Panic(msg string, args ...interface{})

	// Debugf logs a formatted message at DebugLevel
	// Debugf 在DebugLevel级别记录格式化消息
	Debugf(format string, args ...interface{})

	// Infof logs a formatted message at InfoLevel
	// Infof 在InfoLevel级别记录格式化消息
	Infof(format string, args ...interface{})

	// Warnf logs a formatted message at WarnLevel
	// Warnf 在WarnLevel级别记录格式化消息
	Warnf(format string, args ...interface{})

	// Errorf logs a formatted message at ErrorLevel
	// Errorf 在ErrorLevel级别记录格式化消息
	Errorf(format string, args ...interface{})

	// Fatalf logs a formatted message at FatalLevel and then calls os.Exit(1)
	// Fatalf 在FatalLevel级别记录格式化消息，然后调用os.Exit(1)
	Fatalf(format string, args ...interface{})

	// Panicf logs a formatted message at PanicLevel and then panics
	// Panicf 在PanicLevel级别记录格式化消息，然后触发panic
	Panicf(format string, args ...interface{})

	// Log logs a message at the specified level with additional fields
	// Log 在指定级别记录消息，并附加额外的字段
	Log(context context.Context, level LogLevel, msg string, fields Fields)

	// LogWithDuration logs a message with the duration since the start time
	// LogWithDuration 记录一条消息，包含从开始时间起的持续时间
	LogWithDuration(level LogLevel, msg string, start time.Time, fields Fields)

	// ShouldLog returns whether logging is enabled for the specified level
	// ShouldLog 返回是否为指定级别启用日志记录
	ShouldLog(level LogLevel) bool

	// SetLevel sets the logging level
	// SetLevel 设置日志级别
	SetLevel(level LogLevel)

	// GetLevel returns the current logging level
	// GetLevel 返回当前的日志级别
	GetLevel() LogLevel

	// Sync flushes any buffered log entries
	// Sync 刷新任何缓冲的日志条目
	Sync() error

	// Clone returns a copy of the Logger
	// Clone 返回Logger的副本
	Clone() Logger

	// With returns a new Logger instance with the given key-value pairs added to the logger
	// With 返回一个新的Logger实例，其中包含添加到记录器的给定键值对
	With(keysAndValues ...interface{}) Logger

	// WithFields returns a new Logger instance with the given fields added to the logger
	// WithFields 返回一个新的Logger实例，其中包含添加到记录器的给定字段
	WithFields(fields Fields) Logger

	// WithName returns a new Logger with the specified name
	// WithName 返回一个带有指定名称的新Logger
	WithName(name string) Logger

	// WithTrace returns a new Logger with the specified trace ID
	// WithTrace 返回一个带有指定跟踪ID的新Logger
	WithTrace(traceID string) Logger

	// WithError returns a new Logger with the specified error
	// WithError 返回一个带有指定错误的新Logger
	WithError(err error) Logger
}

// LoggerConfig defines the configuration for creating a new Logger
// LoggerConfig 定义创建新Logger的配置
type LoggerConfig struct {
	// Level sets the minimum level of severity for logged messages
	// Level 设置记录消息的最低严重程度级别
	Level LogLevel

	// Format specifies the output format for log messages (JSON or Text)
	// Format 指定日志消息的输出格式（JSON或文本）
	Format LogFormat

	// Output specifies the writer where log messages will be written
	// If nil, logs will be written to standard output
	// Output 指定日志消息写入的 Writer
	// 如果为 nil，则日志将写入标准输出
	Output io.Writer

	// EnableCaller, if true, adds the file name and line number to log messages
	// EnableCaller 如果为true，将文件名和行号添加到日志消息中
	EnableCaller bool

	// EnableStacktrace, if true, adds a stack trace to Error and Fatal level messages
	// EnableStacktrace 如果为true，将堆栈跟踪添加到Error和Fatal级别的消息中
	EnableStacktrace bool

	// InitialFields specifies a collection of fields to add to all log messages
	// InitialFields 指定要添加到所有日志消息的字段集合
	InitialFields Fields

	// FileConfig specifies the configuration for file logging
	// FileConfig 指定文件日志的配置
	FileConfig *FileLogConfig
}

// FileLogConfig defines configuration for file logging
// FileLogConfig 定义文件日志的配置
type FileLogConfig struct {
	// Enable enables file logging
	// Enable 启用文件日志
	Enable bool

	// Environment specifies the running environment (e.g., development, production, testing)
	// Environment 指定运行环境（例如：development、production、testing）
	Environment string

	// Directory specifies the base directory for log files
	// Directory 指定日志文件的基础目录
	Directory string

	// Filename is the file to write logs to
	// Filename 是写入日志的文件
	Filename string

	// MaxSize is the maximum size in megabytes of the log file before it gets rotated
	// MaxSize 是日志文件在轮转之前的最大大小（以兆字节为单位）
	MaxSize int

	// MaxBackups is the maximum number of old log files to retain
	// MaxBackups 是要保留的旧日志文件的最大数量
	MaxBackups int

	// MaxAge is the maximum number of days to retain old log files
	// MaxAge 是保留旧日志文件的最大天数
	MaxAge int

	// Compress determines if the rotated log files should be compressed
	// Compress 确定是否应压缩轮转的日志文件
	Compress bool

	// UseLocalTime determines if local time is used for log file names
	// UseLocalTime 确定是否使用本地时间作为日志文件名
	UseLocalTime bool
}

// ConfigurableLogger defines an optional interface for loggers that allow changing output and format
// ConfigurableLogger 定义了可选接口，用于允许更改输出和格式的记录器
type ConfigurableLogger interface {
	SetOutput(w io.Writer)
	SetFormat(format LogFormat)
}

// NewLogger 创建一个新的Logger实例，并返回可选接口
func NewLogger(config *LoggerConfig, loggerType string) (Logger, error) {
	switch loggerType {
	case "zap":
		return NewZapLogger(config)
	case "zerolog":
		return NewZerologLogger(config)
	case "slog", "":
		return NewSlogLogger(config)
	default:
		return NewSlogLogger(config)
	}
}

// Example usage of optional interfaces:
func ConfigureLoggerOutputAndFormat(logger Logger, writer io.Writer, format LogFormat) {
	// 类型断言检查日志实例是否实现了 ConfigurableLogger 接口
	if configurableLogger, ok := logger.(ConfigurableLogger); ok {
		configurableLogger.SetOutput(writer)
		configurableLogger.SetFormat(format)
	} else {
		// The logger implementation does not support SetOutput or SetFormat.
		GetLogger().Warn("Logger does not support output or format configuration")
	}
}

// NewDefaultConfig returns a default logger configuration
func NewDefaultConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:            InfoLevel,
		Format:           JSONFormat,
		EnableCaller:     true,
		EnableStacktrace: true,
		FileConfig: &FileLogConfig{
			Enable:       true,
			Environment:  os.Getenv("APP_ENVIRONMENT"),
			Directory:    "logs",
			Filename:     "app.log",
			MaxSize:      100,  // 100MB
			MaxBackups:   30,   // 保留30个备份
			MaxAge:       7,    // 保留7天
			Compress:     true, // 压缩轮转的日志
			UseLocalTime: true, // 使用本地时间
		},
	}
}
