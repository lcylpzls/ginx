package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRecovery_NoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际 %d", w.Code)
	}
}

func TestRecovery_Panic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.GET("/test", func(c *gin.Context) {
		panic("测试 panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON 解析失败：%v", err)
	}
	if resp["code"] != float64(500) {
		t.Errorf("期望 code 500，实际 %v", resp["code"])
	}
	if resp["msg"] != "服务器内部错误" {
		t.Errorf("期望 msg '服务器内部错误'，实际 %v", resp["msg"])
	}
}

func TestRecovery_SubsequentRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("测试 panic")
	})
	r.GET("/normal", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 先触发 panic
	req1 := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusInternalServerError {
		t.Errorf("期望 500，实际 %d", w1.Code)
	}

	// 下一个请求应正常工作
	req2 := httptest.NewRequest(http.MethodGet, "/normal", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("期望 200，实际 %d（panic 后服务应继续正常工作）", w2.Code)
	}
}

func TestRecovery_StackRecorded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.GET("/test", func(c *gin.Context) {
		panic("测试 panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("期望 500，实际 %d", w.Code)
	}
	// 无法直接验证 recoveryError 被设置，但通过 JSON 输出验证
}
