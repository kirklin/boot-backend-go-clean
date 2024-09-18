package logger

import (
	"context"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
	level  zap.AtomicLevel
}

var fieldPool = sync.Pool{
	New: func() interface{} {
		return make([]zap.Field, 0, 5)
	},
}

func NewZapLogger(config *LoggerConfig) (Logger, error) {
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(convertToZapLevel(config.Level))

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if config.Format == JSONFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	writer := zapcore.AddSync(os.Stdout)
	if config.Output != nil {
		writer = zapcore.AddSync(config.Output)
	}

	core := zapcore.NewCore(encoder, writer, atomicLevel)

	zapOptions := []zap.Option{zap.AddCallerSkip(1)} // 添加这行以跳过包装器函数

	if config.EnableCaller {
		zapOptions = append(zapOptions, zap.AddCaller())
	}

	if config.EnableStacktrace {
		zapOptions = append(zapOptions, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	logger := zap.New(core, zapOptions...)

	if len(config.InitialFields) > 0 {
		fields := make([]zap.Field, 0, len(config.InitialFields))
		for k, v := range config.InitialFields {
			fields = append(fields, zap.Any(k, v))
		}
		logger = logger.With(fields...)
	}

	return &zapLogger{
		logger: logger,
		sugar:  logger.Sugar(),
		level:  atomicLevel,
	}, nil
}

func (l *zapLogger) Debug(msg string, args ...interface{}) {
	l.sugar.Debugw(msg, args...)
}

func (l *zapLogger) Info(msg string, args ...interface{}) {
	l.sugar.Infow(msg, args...)
}

func (l *zapLogger) Warn(msg string, args ...interface{}) {
	l.sugar.Warnw(msg, args...)
}

func (l *zapLogger) Error(msg string, args ...interface{}) {
	l.sugar.Errorw(msg, args...)
}

func (l *zapLogger) Fatal(msg string, args ...interface{}) {
	l.sugar.Fatalw(msg, args...)
}

func (l *zapLogger) Panic(msg string, args ...interface{}) {
	l.sugar.Panicw(msg, args...)
}

func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
}

func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
}

func (l *zapLogger) Warnf(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
}

func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
}

func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.sugar.Fatalf(format, args...)
}

func (l *zapLogger) Panicf(format string, args ...interface{}) {
	l.sugar.Panicf(format, args...)
}

func (l *zapLogger) Log(ctx context.Context, level LogLevel, msg string, fields Fields) {
	zapLevel := convertToZapLevel(level)
	if ce := l.logger.Check(zapLevel, msg); ce != nil {
		zapFields := fieldsToZapFields(fields)
		if ctx != nil {
			// 添加上下文相关的字段，如果有的话
			if traceID, ok := ctx.Value("traceID").(string); ok {
				zapFields = append(zapFields, zap.String("traceID", traceID))
			}
			// 可以添加其他上下文相关的字段
		}
		ce.Write(zapFields...)
	}
}

func (l *zapLogger) LogWithDuration(level LogLevel, msg string, start time.Time, fields Fields) {
	duration := time.Since(start)
	if fields == nil {
		fields = make(Fields)
	}
	fields["duration"] = duration.Milliseconds()
	l.Log(context.Background(), level, msg, fields)
}

func (l *zapLogger) ShouldLog(level LogLevel) bool {
	return l.level.Enabled(convertToZapLevel(level))
}

func (l *zapLogger) SetLevel(level LogLevel) {
	l.level.SetLevel(convertToZapLevel(level))
}

func (l *zapLogger) GetLevel() LogLevel {
	zapLevel := l.level.Level()
	switch zapLevel {
	case zapcore.DebugLevel:
		return DebugLevel
	case zapcore.InfoLevel:
		return InfoLevel
	case zapcore.WarnLevel:
		return WarnLevel
	case zapcore.ErrorLevel:
		return ErrorLevel
	case zapcore.FatalLevel:
		return FatalLevel
	case zapcore.PanicLevel:
		return PanicLevel
	default:
		return InfoLevel
	}
}

func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}

func (l *zapLogger) Clone() Logger {
	return &zapLogger{
		logger: l.logger,
		sugar:  l.sugar,
		level:  zap.NewAtomicLevelAt(l.level.Level()),
	}
}

func (l *zapLogger) With(keysAndValues ...interface{}) Logger {
	sugar := l.sugar.With(keysAndValues...)
	return &zapLogger{
		logger: sugar.Desugar(),
		sugar:  sugar,
		level:  l.level,
	}
}

func (l *zapLogger) WithFields(fields Fields) Logger {
	logger := l.logger.With(fieldsToZapFields(fields)...)
	return &zapLogger{
		logger: logger,
		sugar:  logger.Sugar(),
		level:  l.level,
	}
}

func (l *zapLogger) WithName(name string) Logger {
	logger := l.logger.Named(name)
	return &zapLogger{
		logger: logger,
		sugar:  logger.Sugar(),
		level:  l.level,
	}
}

func (l *zapLogger) WithTrace(traceID string) Logger {
	logger := l.logger.With(zap.String("traceID", traceID))
	return &zapLogger{
		logger: logger,
		sugar:  logger.Sugar(),
		level:  l.level,
	}
}

func (l *zapLogger) WithError(err error) Logger {
	logger := l.logger.With(zap.Error(err))
	return &zapLogger{
		logger: logger,
		sugar:  logger.Sugar(),
		level:  l.level,
	}
}

func fieldsToZapFields(fields Fields) []zap.Field {
	zapFields := fieldPool.Get().([]zap.Field)
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	defer func() {
		zapFields = zapFields[:0]
		fieldPool.Put(zapFields)
	}()
	return zapFields
}
