package ginx

import (
	"context"
	"time"
)

// Logger 定义 ginx 的日志接口。
//
// 调用方通过实现此接口注入自定义日志组件（如 Zap、Zerolog）。
// 默认使用 NoopLogger，所有方法为空实现，编译器内联，零分配。
type Logger interface {
	// Debug 记录调试级别日志。
	Debug(ctx context.Context, msg string, fields ...Field)

	// Info 记录信息级别日志。
	Info(ctx context.Context, msg string, fields ...Field)

	// Warn 记录警告级别日志。
	Warn(ctx context.Context, msg string, fields ...Field)

	// Error 记录错误级别日志。
	Error(ctx context.Context, msg string, fields ...Field)

	// Fatal 记录致命级别日志。
	Fatal(ctx context.Context, msg string, fields ...Field)
}

// Field 表示一条结构化的日志字段。
type Field struct {
	Key   string
	Value any
}

// StringField 创建一个字符串类型的日志字段。
func StringField(key, val string) Field {
	return Field{Key: key, Value: val}
}

// IntField 创建一个整数类型的日志字段。
func IntField(key string, val int) Field {
	return Field{Key: key, Value: val}
}

// DurationField 创建一个时间间隔类型的日志字段。
func DurationField(key string, val time.Duration) Field {
	return Field{Key: key, Value: val}
}

// ErrorField 创建一个错误类型的日志字段，Key 固定为 "error"。
func ErrorField(err error) Field {
	return Field{Key: "error", Value: err}
}

// AnyField 创建一个任意类型的日志字段。
func AnyField(key string, val any) Field {
	return Field{Key: key, Value: val}
}

// NoopLogger 是 Logger 接口的空实现，所有方法不执行任何操作。
//
// 编译器会将 NoopLogger 的方法内联，确保零分配开销。
// 当调用方未注入自定义 Logger 时，Server 默认使用此实现。
type NoopLogger struct{}

// Debug 空实现。
func (n NoopLogger) Debug(_ context.Context, _ string, _ ...Field) {}

// Info 空实现。
func (n NoopLogger) Info(_ context.Context, _ string, _ ...Field) {}

// Warn 空实现。
func (n NoopLogger) Warn(_ context.Context, _ string, _ ...Field) {}

// Error 空实现。
func (n NoopLogger) Error(_ context.Context, _ string, _ ...Field) {}

// Fatal 空实现。
func (n NoopLogger) Fatal(_ context.Context, _ string, _ ...Field) {}

// 编译期验证 NoopLogger 实现了 Logger 接口。
var _ Logger = NoopLogger{}
