package ginx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStringField(t *testing.T) {
	f := StringField("key1", "value1")
	if f.Key != "key1" {
		t.Errorf("期望 Key 为 'key1'，实际为 %q", f.Key)
	}
	if f.Value != "value1" {
		t.Errorf("期望 Value 为 'value1'，实际为 %v", f.Value)
	}
}

func TestIntField(t *testing.T) {
	f := IntField("count", 42)
	if f.Key != "count" {
		t.Errorf("期望 Key 为 'count'，实际为 %q", f.Key)
	}
	if f.Value != 42 {
		t.Errorf("期望 Value 为 42，实际为 %v", f.Value)
	}
}

func TestDurationField(t *testing.T) {
	d := 5 * time.Second
	f := DurationField("timeout", d)
	if f.Key != "timeout" {
		t.Errorf("期望 Key 为 'timeout'，实际为 %q", f.Key)
	}
	if f.Value != d {
		t.Errorf("期望 Value 为 %v，实际为 %v", d, f.Value)
	}
}

func TestErrorField(t *testing.T) {
	err := errors.New("测试错误")
	f := ErrorField(err)
	if f.Key != "error" {
		t.Errorf("期望 Key 为 'error'，实际为 %q", f.Key)
	}
	if f.Value != err {
		t.Errorf("期望 Value 为 %v，实际为 %v", err, f.Value)
	}
}

func TestErrorField_Nil(t *testing.T) {
	f := ErrorField(nil)
	if f.Key != "error" {
		t.Errorf("期望 Key 为 'error'，实际为 %q", f.Key)
	}
	if f.Value != nil {
		t.Errorf("期望 Value 为 nil，实际为 %v", f.Value)
	}
}

func TestAnyField(t *testing.T) {
	type testStruct struct {
		Name string
	}
	val := testStruct{Name: "test"}
	f := AnyField("data", val)
	if f.Key != "data" {
		t.Errorf("期望 Key 为 'data'，实际为 %q", f.Key)
	}
	// 类型断言恢复
	if v, ok := f.Value.(testStruct); !ok || v.Name != "test" {
		t.Errorf("期望 Value 为 %v，实际为 %v", val, f.Value)
	}
}

func TestNoopLogger_Debug(t *testing.T) {
	var nl NoopLogger
	// 不应 panic
	nl.Debug(context.Background(), "测试消息")
}

func TestNoopLogger_Info(t *testing.T) {
	var nl NoopLogger
	nl.Info(context.Background(), "测试消息", StringField("key", "val"))
}

func TestNoopLogger_Warn(t *testing.T) {
	var nl NoopLogger
	nl.Warn(context.Background(), "测试消息")
}

func TestNoopLogger_Error(t *testing.T) {
	var nl NoopLogger
	nl.Error(context.Background(), "测试消息", ErrorField(errors.New("err")))
}

func TestNoopLogger_Fatal(t *testing.T) {
	var nl NoopLogger
	nl.Fatal(context.Background(), "测试消息")
}

// TestNoopLogger_ImplementsLogger 编译期已通过 var _ Logger = NoopLogger{} 验证。
// 此处额外通过反射验证。
func TestNoopLogger_ImplementsLogger(t *testing.T) {
	var nl NoopLogger
	var l Logger = nl
	_ = l // 仅验证类型兼容
}
