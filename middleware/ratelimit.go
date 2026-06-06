package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 实现基于 IP 的令牌桶限流。
type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*tokenBucket
	qps       int
	window    time.Duration
	whitelist []*net.IPNet
}

type tokenBucket struct {
	tokens   float64
	lastTime time.Time
}

// NewRateLimiter 创建一个新的 IP 限流器。
//
// qps 为每秒允许的请求数，window 为限流窗口，whitelist 为白名单 CIDR 列表。
func NewRateLimiter(qps int, window time.Duration, whitelistCIDRs []string) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		qps:     qps,
		window:  window,
	}

	for _, cidr := range whitelistCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			// 尝试解析为单 IP
			ip := net.ParseIP(cidr)
			if ip != nil {
				_, ipNet, _ = net.ParseCIDR(cidr + "/32")
			}
		}
		if ipNet != nil {
			rl.whitelist = append(rl.whitelist, ipNet)
		}
	}

	return rl
}

// Allow 检查指定 IP 是否被允许通过。
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 检查白名单
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil {
		for _, wl := range rl.whitelist {
			if wl.Contains(parsedIP) {
				return true
			}
		}
	}

	now := time.Now()
	bucket, exists := rl.buckets[ip]
	if !exists {
		bucket = &tokenBucket{
			tokens:   float64(rl.qps),
			lastTime: now,
		}
		rl.buckets[ip] = bucket
	}

	// 计算新增令牌
	elapsed := now.Sub(bucket.lastTime).Seconds()
	bucket.tokens += elapsed * float64(rl.qps)
	if bucket.tokens > float64(rl.qps) {
		bucket.tokens = float64(rl.qps)
	}
	bucket.lastTime = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}
	return false
}

// Cleanup 清理过期的桶。
func (rl *RateLimiter) Cleanup(interval time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, bucket := range rl.buckets {
		if now.Sub(bucket.lastTime) > interval*10 {
			delete(rl.buckets, ip)
		}
	}
}

// extractClientIP 从请求中提取客户端 IP。
//
// 优先级：X-Forwarded-For → X-Real-IP → RemoteAddr
func extractClientIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// 取第一个 IP（可能有代理链）
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

// RateLimit 返回一个 IP 令牌桶限流中间件。
//
// 超限返回标准化 429 响应，白名单 IP/CIDR 直接放行。
func RateLimit(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := extractClientIP(c)

		if !rl.Allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":      429,
				"msg":       "请求过于频繁，请稍后重试",
				"data":      nil,
				"requestId": c.GetString("requestId"),
				"timestamp": time.Now().UnixMilli(),
			})
			return
		}

		c.Next()
	}
}
