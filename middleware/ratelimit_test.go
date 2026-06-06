package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, time.Second, nil)
	if rl.qps != 10 {
		t.Errorf("期望 qps 为 10，实际 %d", rl.qps)
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(100, time.Second, nil)

	// 前 100 次请求应通过
	for i := 0; i < 100; i++ {
		if !rl.Allow("192.168.1.1") {
			t.Errorf("第 %d 次请求应通过", i+1)
		}
	}

	// 第 101 次应被拒绝
	if rl.Allow("192.168.1.1") {
		t.Error("第 101 次请求应被限流")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := NewRateLimiter(5, time.Second, nil)

	// IP1 用完额度
	for i := 0; i < 5; i++ {
		rl.Allow("10.0.0.1")
	}
	if rl.Allow("10.0.0.1") {
		t.Error("IP1 应被限流")
	}

	// IP2 应有独立额度
	if !rl.Allow("10.0.0.2") {
		t.Error("IP2 应能通过")
	}
}

func TestRateLimiter_Whitelist(t *testing.T) {
	rl := NewRateLimiter(1, time.Second, []string{"10.0.0.0/8"})

	// 白名单 IP 始终通过
	for i := 0; i < 100; i++ {
		if !rl.Allow("10.0.0.1") {
			t.Errorf("白名单 IP 第 %d 次应通过", i+1)
		}
	}

	// 非白名单 IP 受限
	rl2 := NewRateLimiter(1, time.Second, []string{"10.0.0.0/8"})
	if !rl2.Allow("192.168.1.1") {
		t.Error("第 1 次应通过")
	}
	if rl2.Allow("192.168.1.1") {
		t.Error("第 2 次应被限流")
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(100, time.Second, nil)
	rl.Allow("192.168.1.1")

	if len(rl.buckets) != 1 {
		t.Errorf("期望 1 个桶，实际 %d", len(rl.buckets))
	}

	// 清理无法直接验证（依赖时间），但至少确保不 panic
	rl.Cleanup(time.Minute)
}

func TestRateLimitMiddleware_Normal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(100, time.Second, nil)
	r := gin.New()
	r.Use(RateLimit(rl))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，实际 %d", w.Code)
	}
}

func TestRateLimitMiddleware_Exceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(1, time.Second, nil)
	r := gin.New()
	r.Use(RateLimit(rl))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 第一次通过
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("第一次期望 200，实际 %d", w1.Code)
	}

	// 第二次被限流
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("期望 429，实际 %d", w2.Code)
	}
}

func TestRateLimitMiddleware_WhitelistByIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRateLimiter(1, time.Second, []string{"192.168.1.0/24"})
	r := gin.New()
	r.Use(RateLimit(rl))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 白名单 IP 可连续请求
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("白名单 IP 第 %d 次期望 200，实际 %d", i+1, w.Code)
		}
	}
}

func TestExtractClientIP_XForwardedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		ip := extractClientIP(c)
		c.String(http.StatusOK, ip)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != "10.0.0.1" {
		t.Errorf("期望 '10.0.0.1'，实际 %q", w.Body.String())
	}
}

func TestExtractClientIP_XRealIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		ip := extractClientIP(c)
		c.String(http.StatusOK, ip)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Real-IP", "10.0.0.2")
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != "10.0.0.2" {
		t.Errorf("期望 '10.0.0.2'，实际 %q", w.Body.String())
	}
}

func TestExtractClientIP_RemoteAddr(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		ip := extractClientIP(c)
		c.String(http.StatusOK, ip)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != "192.168.1.1" {
		t.Errorf("期望 '192.168.1.1'，实际 %q", w.Body.String())
	}
}

func TestExtractClientIP_XForwardedForPriority(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		ip := extractClientIP(c)
		c.String(http.StatusOK, ip)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.Header.Set("X-Real-IP", "10.0.0.2")
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// X-Forwarded-For 优先级最高
	if w.Body.String() != "10.0.0.1" {
		t.Errorf("期望 '10.0.0.1'，实际 %q", w.Body.String())
	}
}

func TestRateLimiter_Allow_EmptyWhitelist(t *testing.T) {
	// 无白名单时，有效 IP 走令牌桶逻辑
	rl := NewRateLimiter(100, time.Second, nil)
	if !rl.Allow("192.168.1.1") {
		t.Error("期望 Allow 返回 true（无白名单，首次请求）")
	}
}

func TestRateLimiter_Allow_TokenRefill(t *testing.T) {
	// 测试令牌超过 QPS 上限时的截断逻辑
	rl := NewRateLimiter(1, time.Second, nil)
	// 第一次通过，消耗 1 个令牌
	if !rl.Allow("192.168.1.1") {
		t.Fatal("首次应通过")
	}
	// 等待足够时间让令牌超过 QPS 上限
	time.Sleep(2 * time.Second)
	// 经过 2 秒，令牌应恢复但不超过 QPS(1)
	if !rl.Allow("192.168.1.1") {
		t.Error("令牌恢复后应能通过")
	}
}

func TestRateLimiter_Cleanup_Expired(t *testing.T) {
	rl := NewRateLimiter(100, time.Second, nil)
	rl.Allow("192.168.1.1")

	if len(rl.buckets) != 1 {
		t.Fatalf("期望 1 个桶，实际 %d", len(rl.buckets))
	}

	// 等待一小段时间后使用零间隔清理
	time.Sleep(time.Millisecond)
	rl.Cleanup(0)
	if len(rl.buckets) != 0 {
		t.Errorf("期望桶被清理，实际剩余 %d 个", len(rl.buckets))
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(1000, time.Second, nil)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				rl.Allow("10.0.0.1")
			}
		}()
	}
	wg.Wait()
	// 仅验证无 race condition panic
}

func TestNewRateLimiter_InvalidWhitelistEntry(t *testing.T) {
	// 无效的 CIDR 且无法解析为单 IP，应被跳过
	rl := NewRateLimiter(10, time.Second, []string{"not-a-valid-cidr-or-ip"})
	if len(rl.whitelist) != 0 {
		t.Errorf("期望 0 个白名单条目，实际 %d", len(rl.whitelist))
	}
}

func TestNewRateLimiter_SingleIPWhitelist(t *testing.T) {
	// 单个 IP 应被转换为 /32 CIDR
	rl := NewRateLimiter(10, time.Second, []string{"10.0.0.1"})
	if len(rl.whitelist) != 1 {
		t.Fatalf("期望 1 个白名单条目，实际 %d", len(rl.whitelist))
	}
	// 验证该 IP 在白名单中
	if !rl.Allow("10.0.0.1") {
		t.Error("白名单 IP 应被允许")
	}
}

func TestRateLimiter_Allow_InvalidIP(t *testing.T) {
	rl := NewRateLimiter(1, time.Second, []string{"10.0.0.0/8"})
	// 无效 IP 不能解析，应跳过白名单检查，走令牌桶逻辑
	result := rl.Allow("invalid-ip")
	// 应该按正常令牌桶处理（不 panic），首次应通过
	if !result {
		t.Error("无效 IP 首次请求应通过")
	}
	// 第二次应被限流（无效 IP 也走令牌桶）
	result = rl.Allow("invalid-ip")
	if result {
		t.Error("无效 IP 第二次请求应被限流")
	}
}

func TestRateLimiter_Cleanup_NoExpired(t *testing.T) {
	rl := NewRateLimiter(100, time.Second, nil)
	rl.Allow("192.168.1.1")

	if len(rl.buckets) != 1 {
		t.Errorf("期望 1 个桶，实际 %d", len(rl.buckets))
	}

	// 使用很长的间隔，桶不应被清理
	rl.Cleanup(time.Hour)
	if len(rl.buckets) != 1 {
		t.Errorf("期望桶未被清理，实际 %d", len(rl.buckets))
	}
}

func TestExtractClientIP_NoPort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		ip := extractClientIP(c)
		c.String(http.StatusOK, ip)
	})

	// RemoteAddr 不含端口号
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 不含端口时 SplitHostPort 返回错误，应直接返回 RemoteAddr
	if w.Body.String() != "192.168.1.1" {
		t.Errorf("期望 '192.168.1.1'，实际 %q", w.Body.String())
	}
}
