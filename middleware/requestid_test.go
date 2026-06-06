package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID_Generate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		id, exists := c.Get("requestId")
		if !exists {
			t.Error("期望 requestId 存在于 Context 中")
		}
		if idStr, ok := id.(string); !ok || idStr == "" {
			t.Error("期望 requestId 为非空字符串")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	respID := w.Header().Get("X-Request-ID")
	if respID == "" {
		t.Error("期望响应头 X-Request-ID 非空")
	}
}

func TestRequestID_ReuseHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		id, _ := c.Get("requestId")
		if id != "my-custom-id" {
			t.Errorf("期望 requestId 为 'my-custom-id'，实际 %v", id)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "my-custom-id")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	respID := w.Header().Get("X-Request-ID")
	if respID != "my-custom-id" {
		t.Errorf("期望响应头 X-Request-ID 为 'my-custom-id'，实际 %q", respID)
	}
}

func TestRequestID_UUIDv4Format(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		id, _ := c.Get("requestId")
		idStr := id.(string)
		// UUID v4 格式：36 个字符（含 4 个连字符）
		if len(idStr) != 36 {
			t.Errorf("期望 UUID v4 长度为 36，实际 %d (%s)", len(idStr), idStr)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	_ = w
}
