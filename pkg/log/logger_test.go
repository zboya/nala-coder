package log

import (
	"testing"

	"github.com/spf13/viper"
)

func TestNewLogrusLogger(t *testing.T) {
	config := &Config{
		Level:  "info",
		Output: "stdout",
		Format: "text",
	}

	logger, err := NewLogrusLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Logger is nil")
	}

	// 测试基本日志方法
	logger.Info("Test info message")
	logger.Debug("Test debug message")
	logger.Error("Test error message")
}

func TestNewFromViper(t *testing.T) {
	// 设置测试配置
	viper.Set("logging.level", "debug")
	viper.Set("logging.output", "stdout")
	viper.Set("logging.format", "json")

	logger, err := NewFromViper()
	if err != nil {
		t.Fatalf("Failed to create logger from viper: %v", err)
	}

	if logger == nil {
		t.Fatal("Logger is nil")
	}

	// 测试日志输出
	logger.Info("Test viper logger")
}

func TestGetLogrusInstance(t *testing.T) {
	config := &Config{
		Level:  "info",
		Output: "stdout",
		Format: "text",
	}

	logger, err := NewLogrusLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logrusInstance := GetLogrusInstance(logger)
	if logrusInstance == nil {
		t.Fatal("Logrus instance is nil")
	}

	// 测试logrus实例
	logrusInstance.Info("Test logrus instance")
}

func TestLevelParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DebugLevel},
		{"info", InfoLevel},
		{"warn", WarnLevel},
		{"error", ErrorLevel},
		{"fatal", FatalLevel},
		{"panic", PanicLevel},
		{"invalid", InfoLevel}, // 默认值
	}

	for _, tt := range tests {
		level, _ := ParseLevel(tt.input)
		if level != tt.expected {
			t.Errorf("ParseLevel(%s) = %v, want %v", tt.input, level, tt.expected)
		}
	}
}

func TestConfigValidation(t *testing.T) {
	config := &Config{
		Level:  "info",
		Output: "stdout",
		Format: "text",
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Valid config should not return error: %v", err)
	}

	// 测试无效配置会被修复
	config.Output = "invalid"
	config.Format = "invalid"
	err = config.Validate()
	if err != nil {
		t.Errorf("Config validation should fix invalid values: %v", err)
	}

	if config.Output != "stdout" {
		t.Errorf("Invalid output should be fixed to stdout, got: %s", config.Output)
	}

	if config.Format != "text" {
		t.Errorf("Invalid format should be fixed to text, got: %s", config.Format)
	}
}
