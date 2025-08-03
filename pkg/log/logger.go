package log

import (
	"context"
	"io"
)

// Logger 定义日志接口，方便扩展其他log实现
type Logger interface {
	// 基础日志方法
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Fatal(args ...any)
	Panic(args ...any)

	// 格式化日志方法
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Panicf(format string, args ...any)

	// 日志级别设置
	SetLevel(level Level)
	GetLevel() Level

	// 输出设置
	SetOutput(output io.Writer)
	GetOutput() io.Writer

	// 格式设置
	SetFormatter(formatter Formatter)

	// 上下文相关
	WithContext(ctx context.Context) Logger
	WithField(key string, value any) Logger
	WithFields(fields Fields) Logger
}

// Level 日志级别
type Level int

const (
	PanicLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	TraceLevel
)

// String 返回日志级别字符串
func (level Level) String() string {
	switch level {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	case TraceLevel:
		return "trace"
	default:
		return "unknown"
	}
}

// ParseLevel 解析日志级别字符串
func ParseLevel(lvl string) (Level, error) {
	switch lvl {
	case "panic":
		return PanicLevel, nil
	case "fatal":
		return FatalLevel, nil
	case "error":
		return ErrorLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	case "trace":
		return TraceLevel, nil
	default:
		return InfoLevel, nil
	}
}

// Fields 日志字段类型
type Fields map[string]any

// Formatter 格式化器接口
type Formatter interface {
	Format(level Level, msg string, fields Fields) ([]byte, error)
}
