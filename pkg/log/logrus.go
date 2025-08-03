package log

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// LogrusLogger 基于logrus的Logger实现
type LogrusLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

// NewLogrusLogger 创建基于logrus的Logger
func NewLogrusLogger(config *Config) (Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger := logrus.New()

	// 设置日志级别
	level, err := ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	logger.SetLevel(logrus.Level(level))

	// 设置输出
	if err := setLogOutput(logger, config); err != nil {
		return nil, fmt.Errorf("failed to set log output: %w", err)
	}

	// 设置格式
	setLogFormatter(logger, config)

	return &LogrusLogger{
		logger: logger,
		entry:  logrus.NewEntry(logger),
	}, nil
}

// setLogOutput 设置日志输出
func setLogOutput(logger *logrus.Logger, config *Config) error {
	var writers []io.Writer

	switch config.Output {
	case "file":
		// 只输出到文件
		file, err := openLogFile(config.File)
		if err != nil {
			fmt.Printf("Failed to open log file %s: %v, falling back to stdout\n", config.File, err)
			writers = append(writers, os.Stdout)
		} else {
			writers = append(writers, file)
		}
	case "both":
		// 同时输出到文件和stdout
		writers = append(writers, os.Stdout)
		if file, err := openLogFile(config.File); err == nil {
			writers = append(writers, file)
		} else {
			fmt.Printf("Failed to open log file %s: %v, only logging to stdout\n", config.File, err)
		}
	default:
		// 默认输出到stdout
		writers = append(writers, os.Stdout)
	}

	if len(writers) > 1 {
		logger.SetOutput(io.MultiWriter(writers...))
	} else {
		logger.SetOutput(writers[0])
	}

	return nil
}

// setLogFormatter 设置日志格式
func setLogFormatter(logger *logrus.Logger, config *Config) {
	if config.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			DisableQuote:  true,
		})
	}
}

// openLogFile 打开日志文件
func openLogFile(logFile string) (*os.File, error) {
	if logFile == "" {
		return nil, fmt.Errorf("log file path is empty")
	}

	// 创建日志目录
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// 打开日志文件（追加模式）
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", logFile, err)
	}

	return file, nil
}

// 基础日志方法
func (l *LogrusLogger) Debug(args ...interface{}) {
	l.entry.Debug(args...)
}

func (l *LogrusLogger) Info(args ...interface{}) {
	l.entry.Info(args...)
}

func (l *LogrusLogger) Warn(args ...interface{}) {
	l.entry.Warn(args...)
}

func (l *LogrusLogger) Error(args ...interface{}) {
	l.entry.Error(args...)
}

func (l *LogrusLogger) Fatal(args ...interface{}) {
	l.entry.Fatal(args...)
}

func (l *LogrusLogger) Panic(args ...interface{}) {
	l.entry.Panic(args...)
}

// 格式化日志方法
func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}

func (l *LogrusLogger) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

func (l *LogrusLogger) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}

func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

func (l *LogrusLogger) Fatalf(format string, args ...interface{}) {
	l.entry.Fatalf(format, args...)
}

func (l *LogrusLogger) Panicf(format string, args ...interface{}) {
	l.entry.Panicf(format, args...)
}

// 日志级别设置
func (l *LogrusLogger) SetLevel(level Level) {
	l.logger.SetLevel(logrus.Level(level))
}

func (l *LogrusLogger) GetLevel() Level {
	return Level(l.logger.GetLevel())
}

// 输出设置
func (l *LogrusLogger) SetOutput(output io.Writer) {
	l.logger.SetOutput(output)
}

func (l *LogrusLogger) GetOutput() io.Writer {
	return l.logger.Out
}

// 格式设置
func (l *LogrusLogger) SetFormatter(formatter Formatter) {
	// 这里可以实现自定义格式化器，暂时留空
}

// 上下文相关
func (l *LogrusLogger) WithContext(ctx context.Context) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithContext(ctx),
	}
}

func (l *LogrusLogger) WithField(key string, value interface{}) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
	}
}

func (l *LogrusLogger) WithFields(fields Fields) Logger {
	return &LogrusLogger{
		logger: l.logger,
		entry:  l.entry.WithFields(logrus.Fields(fields)),
	}
}

// GetLogrusInstance 获取底层的logrus实例（用于向后兼容）
func (l *LogrusLogger) GetLogrusInstance() *logrus.Logger {
	return l.logger
}
