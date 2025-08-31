package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// TestNewLogger tests the creation of different logger types
func TestNewLogger(t *testing.T) {
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
	}

	tests := []struct {
		name       string
		loggerType string
		wantErr    bool
	}{
		{"Zap Logger", "zap", false},
		{"Zerolog Logger", "zerolog", false},
		{"Slog Logger", "slog", false},
		{"Default Logger", "", false},
		{"Invalid Logger", "invalid", false}, // Should default to slog
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLogger(config, tt.loggerType)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

// TestLoggerMethods tests various logging methods
func TestLoggerMethods(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  DebugLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, err := NewLogger(config, "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	tests := []struct {
		name      string
		logFunc   func()
		wantLevel string
	}{
		{"Debug", func() { logger.Debug("debug message") }, "DEBUG"},
		{"Info", func() { logger.Info("info message") }, "INFO"},
		{"Warn", func() { logger.Warn("warn message") }, "WARN"},
		{"Error", func() { logger.Error("error message") }, "ERROR"},
		{"Debugf", func() { logger.Debugf("debug %s", "formatted") }, "DEBUG"},
		{"Infof", func() { logger.Infof("info %s", "formatted") }, "INFO"},
		{"Warnf", func() { logger.Warnf("warn %s", "formatted") }, "WARN"},
		{"Errorf", func() { logger.Errorf("error %s", "formatted") }, "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("Failed to unmarshal log entry: %v", err)
			}

			if level, ok := logEntry["level"]; !ok || level != tt.wantLevel {
				t.Errorf("Incorrect log level. Got %v, want %v", level, tt.wantLevel)
			}
		})
	}
}

// TestLoggerWithFields tests logging with additional fields
func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewLogger(config, "")

	fields := Fields{"key1": "value1", "key2": 42}
	logger.WithFields(fields).Info("test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if logEntry["key1"] != "value1" || logEntry["key2"] != float64(42) {
		t.Errorf("Log entry does not contain expected fields")
	}
}

// TestLoggerSetLevel tests changing the log level
func TestLoggerSetLevel(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewLogger(config, "")

	logger.Debug("should not be logged")
	if buf.Len() > 0 {
		t.Errorf("Debug message was logged when it shouldn't have been")
	}

	logger.SetLevel(DebugLevel)
	logger.Debug("should be logged")
	if buf.Len() == 0 {
		t.Errorf("Debug message was not logged when it should have been")
	}
}

// TestLoggerLogWithDuration tests logging with duration
func TestLoggerLogWithDuration(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewLogger(config, "")

	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	logger.LogWithDuration(InfoLevel, "test message", start, nil)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if duration, ok := logEntry["duration"]; !ok || duration.(float64) < 10 {
		t.Errorf("Log entry does not contain expected duration")
	}
}

// TestLoggerWithName tests logging with a name
func TestLoggerWithName(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewLogger(config, "")

	logger.WithName("TestLogger").Info("test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if logEntry["logger"] != "TestLogger" {
		t.Errorf("Log entry does not contain expected logger name")
	}
}

// TestLoggerWithError tests logging with an error
func TestLoggerWithError(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewLogger(config, "")

	err := errors.New("test error")
	logger.WithError(err).Error("error occurred")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if logEntry["error"] != "test error" {
		t.Errorf("Log entry does not contain expected error message")
	}
}

// TestLoggerShouldLog tests the ShouldLog method
func TestLoggerShouldLog(t *testing.T) {
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
	}

	logger, _ := NewLogger(config, "")

	if logger.ShouldLog(DebugLevel) {
		t.Errorf("ShouldLog returned true for DebugLevel when it should be false")
	}

	if !logger.ShouldLog(InfoLevel) {
		t.Errorf("ShouldLog returned false for InfoLevel when it should be true")
	}
}

// TestLoggerGetLevel tests the GetLevel method
func TestLoggerGetLevel(t *testing.T) {
	config := &LoggerConfig{
		Level:  WarnLevel,
		Format: JSONFormat,
	}

	logger, _ := NewLogger(config, "")

	if logger.GetLevel() != WarnLevel {
		t.Errorf("GetLevel returned %v, expected %v", logger.GetLevel(), WarnLevel)
	}
}

// TestLoggerSetFormat tests changing the log format
func TestLoggerSetFormat(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewLogger(config, "")

	logger.Info("JSON format")
	if !json.Valid(buf.Bytes()) {
		t.Errorf("Log output is not valid JSON")
	}

	buf.Reset()
	// 类型断言检查日志实例是否实现了 ConfigurableLogger 接口
	if configurableLogger, ok := logger.(ConfigurableLogger); ok {
		configurableLogger.SetFormat(TextFormat)
		logger.Info("Text format")
		if json.Valid(buf.Bytes()) {
			t.Errorf("Log output is JSON when it should be text")
		}
	} else {
		// The logger implementation does not support SetOutput or SetFormat.
		t.Log("Logger does not support output or format configuration")
	}
}

// TestLoggerWithTrace tests logging with a trace ID
func TestLoggerWithTrace(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewLogger(config, "")

	logger.WithTrace("trace-123").Info("test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if logEntry["traceID"] != "trace-123" {
		t.Errorf("Log entry does not contain expected trace ID")
	}
}

// TestLoggerWith tests logging with additional key-value pairs
func TestLoggerWith(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewLogger(config, "")

	logger.With("key1", "value1", "key2", 42).Info("test message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if logEntry["key1"] != "value1" || logEntry["key2"] != float64(42) {
		t.Errorf("Log entry does not contain expected key-value pairs")
	}
}

// TestLoggerClone tests cloning a logger
func TestLoggerClone(t *testing.T) {
	config := &LoggerConfig{
		Level:  ErrorLevel,
		Format: JSONFormat,
	}

	originalLogger, _ := NewLogger(config, "")
	clonedLogger := originalLogger.Clone()

	if clonedLogger.GetLevel() != originalLogger.GetLevel() {
		t.Errorf("Cloned logger has different level than original")
	}
}

// TestLoggerPanic tests the Panic logging method
func TestLoggerPanic(t *testing.T) {
	var buf bytes.Buffer
	config := &LoggerConfig{
		Level:  InfoLevel,
		Format: JSONFormat,
		Output: &buf,
	}

	logger, _ := NewLogger(config, "")

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	logger.Panic("panic message")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if logEntry["level"] != "PANIC" || logEntry["message"] != "panic message" {
		t.Errorf("Log entry does not contain expected panic message")
	}
}
