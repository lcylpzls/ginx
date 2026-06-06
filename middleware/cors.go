package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig 定义 CORS 中间件的配置参数。
type CORSConfig struct {
	// AllowedOrigins 允许的来源列表，"*" 表示允许所有来源。
	AllowedOrigins []string

	// AllowedMethods 允许的 HTTP 方法列表。
	AllowedMethods []string

	// AllowedHeaders 允许的请求头列表。
	AllowedHeaders []string

	// MaxAge 预检请求缓存时间（秒）。
	MaxAge int
}

// CORS 返回一个 CORS 跨域处理中间件。
//
// 根据配置自动设置 CORS 响应头，处理 OPTIONS 预检请求。
// 若请求 Origin 不在允许列表中，返回 403。
func CORS(cfg CORSConfig) gin.HandlerFunc {
	allowAllOrigins := false
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			allowAllOrigins = true
			break
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 设置允许的来源
		if allowAllOrigins {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			allowed := false
			for _, o := range cfg.AllowedOrigins {
				if o == origin {
					allowed = true
					break
				}
			}
			if allowed {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}

		// 设置允许的方法
		if len(cfg.AllowedMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
		}

		// 设置允许的请求头
		if len(cfg.AllowedHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
		}

		// 设置 MaxAge
		if cfg.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
		}

		// 处理 OPTIONS 预检请求
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// DefaultCORSConfig 返回一组常用的 CORS 默认配置。
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		MaxAge:         86400,
	}
}
