package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTest() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func dummyHandler(msg string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusOK, msg)
	}
}

func dummyMiddleware(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(name, true)
		c.Next()
	}
}

func TestNewManager_DefaultState(t *testing.T) {
	m := NewManager()
	ctx := context.Background()
	chain := m.Build(ctx)

	// RateLimit 默认禁用，所以链中不应包含它
	// 其他中间件默认启用但尚未注册 builtin，所以链为空
	if len(chain) != 0 {
		t.Errorf("期望 0 个中间件（均未注册 builtin），实际 %d", len(chain))
	}
}

func TestManager_RegisterBuiltin(t *testing.T) {
	m := NewManager()
	mw := dummyMiddleware("test")

	m.RegisterBuiltin("request_id", mw)
	chain := m.Build(context.Background())

	if len(chain) != 1 {
		t.Fatalf("期望 1 个中间件，实际 %d", len(chain))
	}
}

func TestManager_Build_Order(t *testing.T) {
	m := NewManager()

	// 注册所有中间件
	m.RegisterBuiltin("recovery", dummyMiddleware("recovery"))
	m.RegisterBuiltin("request_id", dummyMiddleware("request_id"))
	m.RegisterBuiltin("timeout", dummyMiddleware("timeout"))
	m.RegisterBuiltin("cors", dummyMiddleware("cors"))
	m.RegisterBuiltin("validation", dummyMiddleware("validation"))

	chain := m.Build(context.Background())

	// RateLimit 默认禁用，所以应有 5 个
	if len(chain) != 5 {
		t.Fatalf("期望 5 个中间件，实际 %d", len(chain))
	}
}

func TestManager_Disable(t *testing.T) {
	m := NewManager()

	m.RegisterBuiltin("recovery", dummyMiddleware("recovery"))
	m.RegisterBuiltin("request_id", dummyMiddleware("request_id"))
	m.RegisterBuiltin("cors", dummyMiddleware("cors"))

	m.Disable("cors", "recovery")
	chain := m.Build(context.Background())

	// 仅 request_id 未被禁用
	if len(chain) != 1 {
		t.Fatalf("期望 1 个中间件，实际 %d", len(chain))
	}
}

func TestManager_Enable(t *testing.T) {
	m := NewManager()

	m.RegisterBuiltin("recovery", dummyMiddleware("recovery"))
	m.RegisterBuiltin("validation", dummyMiddleware("validation"))

	// 先禁用
	m.Disable("validation")
	chain := m.Build(context.Background())
	if len(chain) != 1 {
		t.Fatalf("禁用后期望 1 个中间件，实际 %d", len(chain))
	}

	// 再启用
	m.Enable("validation")
	chain = m.Build(context.Background())
	if len(chain) != 2 {
		t.Fatalf("启用后期望 2 个中间件，实际 %d", len(chain))
	}
}

func TestManager_EnableRateLimit_Ineffective(t *testing.T) {
	m := NewManager()

	m.RegisterBuiltin("rate_limit", dummyMiddleware("rate_limit"))
	chain := m.Build(context.Background())

	// RateLimit 默认禁用，即使注册了也不会出现在链中
	if len(chain) != 0 {
		t.Fatalf("期望 0 个中间件（RateLimit 默认禁用），实际 %d", len(chain))
	}

	// Enable 对 rate_limit 无效
	m.Enable("rate_limit")
	chain = m.Build(context.Background())
	if len(chain) != 0 {
		t.Fatalf("Enable 对 rate_limit 应无效，实际 %d 个中间件", len(chain))
	}
}

func TestManager_EnableRateLimit(t *testing.T) {
	m := NewManager()

	rlHandler := dummyMiddleware("rate_limit")
	m.EnableRateLimit(rlHandler)

	chain := m.Build(context.Background())
	if len(chain) != 1 {
		t.Fatalf("期望 1 个中间件（rate_limit），实际 %d", len(chain))
	}
}

func TestManager_DisableRateLimit(t *testing.T) {
	m := NewManager()

	rlHandler := dummyMiddleware("rate_limit")
	m.EnableRateLimit(rlHandler)

	chain := m.Build(context.Background())
	if len(chain) != 1 {
		t.Fatalf("期望 1 个中间件，实际 %d", len(chain))
	}

	m.DisableRateLimit()
	chain = m.Build(context.Background())
	if len(chain) != 0 {
		t.Fatalf("DisableRateLimit 后期望 0 个中间件，实际 %d", len(chain))
	}
}

func TestManager_Override(t *testing.T) {
	m := NewManager()
	original := dummyMiddleware("original")
	custom := dummyMiddleware("custom")

	m.RegisterBuiltin("recovery", original)
	m.Override("recovery", custom)

	chain := m.Build(context.Background())
	if len(chain) != 1 {
		t.Fatalf("期望 1 个中间件，实际 %d", len(chain))
	}

	// 验证 custom 被使用：通过 gin 请求验证
	r := setupTest()
	r.Use(m.Build(context.Background())...)
	r.GET("/test", func(c *gin.Context) {
		if _, exists := c.Get("custom"); !exists {
			t.Error("期望 custom 中间件生效")
		}
		if _, exists := c.Get("original"); exists {
			t.Error("不应存在 original 中间件的标记")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
}

func TestManager_Append(t *testing.T) {
	m := NewManager()

	m.RegisterBuiltin("request_id", dummyMiddleware("request_id"))
	m.Append(dummyMiddleware("ext1"), dummyMiddleware("ext2"))

	chain := m.Build(context.Background())
	// request_id + ext1 + ext2 = 3
	if len(chain) != 3 {
		t.Fatalf("期望 3 个中间件，实际 %d", len(chain))
	}
}

func TestManager_Concurrent(t *testing.T) {
	m := NewManager()
	m.RegisterBuiltin("request_id", dummyMiddleware("request_id"))

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				m.Build(context.Background())
				m.Disable("request_id")
				m.Enable("request_id")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
