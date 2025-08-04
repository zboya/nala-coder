package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// expandPath 扩展路径，处理 ~ 符号
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // 如果获取失败，返回原路径
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// New 创建一个新的Logger实例
func New(config *Config) (Logger, error) {
	return NewLogrusLogger(config)
}

// NewFromViper 从viper配置创建Logger
func NewFromViper() (Logger, error) {
	config := &Config{
		Level:  viper.GetString("logging.level"),
		Output: viper.GetString("logging.output"),
		File:   viper.GetString("logging.file"),
		Format: viper.GetString("logging.format"),
	}

	// 如果配置为空，使用默认配置
	if config.Level == "" {
		config = DefaultConfig()
	}

	// 扩展日志文件路径，处理 ~ 符号
	if config.File != "" {
		config.File = expandPath(config.File)
	}

	return New(config)
}

// NewFromViperWithVerbose 从viper配置创建Logger，支持verbose模式
func NewFromViperWithVerbose(verbose bool) (Logger, error) {
	config := &Config{
		Level:  viper.GetString("logging.level"),
		Output: viper.GetString("logging.output"),
		File:   viper.GetString("logging.file"),
		Format: viper.GetString("logging.format"),
	}

	// 如果配置为空，使用默认配置
	if config.Level == "" {
		config = DefaultConfig()
	}

	// 如果verbose模式开启，设置为debug级别
	if verbose {
		config.Level = "debug"
	}

	// 扩展日志文件路径，处理 ~ 符号
	if config.File != "" {
		config.File = expandPath(config.File)
	}

	return New(config)
}

// MustNew 创建Logger，如果失败则panic
func MustNew(config *Config) Logger {
	logger, err := New(config)
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	return logger
}

// MustNewFromViper 从viper配置创建Logger，如果失败则panic
func MustNewFromViper() Logger {
	logger, err := NewFromViper()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger from viper: %v", err))
	}
	return logger
}

// GetLogrusInstance 获取Logger的底层logrus实例（用于向后兼容）
func GetLogrusInstance(logger Logger) *logrus.Logger {
	if logrusLogger, ok := logger.(*LogrusLogger); ok {
		return logrusLogger.GetLogrusInstance()
	}
	panic("logger is not a LogrusLogger instance")
}
