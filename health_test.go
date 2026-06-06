package ginx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestHealthHandler_Response(t *testing.T) {
	gin.SetMode(gin.TestMode)
	startTime := time.Now().Add(-2*time.Hour - 34*time.Minute - 15*time.Second)

	r := gin.New()
	r.GET("/health", healthHandler(startTime))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", w.Code)
	}

	var resp StandardizedResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON 解析失败：%v", err)
	}
	if resp.Code != CodeSuccess {
		t.Errorf("期望 code 0，实际 %d", resp.Code)
	}
	if resp.Msg != "ok" {
		t.Errorf("期望 msg 'ok'，实际 %q", resp.Msg)
	}

	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("期望 Data 为 map，实际类型 %T", resp.Data)
	}
	if dataMap["status"] != "运行中" {
		t.Errorf("期望 status '运行中'，实际 %v", dataMap["status"])
	}
	if dataMap["started"] == "" {
		t.Error("期望 started 非空")
	}
}

func TestHealthHandler_CustomPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	startTime := time.Now()

	r := gin.New()
	r.GET("/custom-health", healthHandler(startTime))

	req := httptest.NewRequest(http.MethodGet, "/custom-health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际 %d", w.Code)
	}
}

func TestHealthHandler_WithRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	startTime := time.Now()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("requestId", "health-test-123")
		c.Next()
	})
	r.GET("/health", healthHandler(startTime))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp StandardizedResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON 解析失败：%v", err)
	}
	if resp.RequestID != "health-test-123" {
		t.Errorf("期望 requestId 'health-test-123'，实际 %q", resp.RequestID)
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"秒级", 30 * time.Second, "30秒"},
		{"分钟级", 5 * time.Minute, "5分钟"},
		{"分钟+秒", 5*time.Minute + 30*time.Second, "5分钟30秒"},
		{"小时级", 2 * time.Hour, "2小时"},
		{"小时+分钟", 2*time.Hour + 30*time.Minute, "2小时30分钟"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatUptime(tt.duration)
			if got != tt.want {
				t.Errorf("formatUptime(%v)=%q，期望 %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestFormatUptime_ZeroSeconds(t *testing.T) {
	got := formatUptime(0)
	if !strings.Contains(got, "秒") {
		t.Errorf("期望包含 '秒'，实际 %q", got)
	}
}
