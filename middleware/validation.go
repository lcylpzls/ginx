package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Validation 返回一个请求参数校验中间件。
//
// 校验 Content-Type 是否为 JSON、Content-Length 是否合理。
// 校验失败返回标准化 400 响应。
func Validation() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 仅对有请求体的方法进行校验
		if c.Request.Method == http.MethodGet ||
			c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		// Content-Type 校验
		contentType := c.GetHeader("Content-Type")
		if contentType == "" {
			// 无 Content-Type 则放行（可能是 GET 类请求或空 Body）
			c.Next()
			return
		}

		// 检查是否为 JSON
		if !isJSONContentType(contentType) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"code":      400,
				"msg":       "请求参数校验失败：Content-Type 必须为 application/json",
				"data":      nil,
				"requestId": c.GetString("requestId"),
				"timestamp": time.Now().UnixMilli(),
			})
			return
		}

		// Content-Length 校验（可选，防止超大请求）
		if c.Request.ContentLength > 10*1024*1024 { // 10MB 默认上限
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"code":      400,
				"msg":       "请求参数校验失败：请求体过大",
				"data":      nil,
				"requestId": c.GetString("requestId"),
				"timestamp": time.Now().UnixMilli(),
			})
			return
		}

		c.Next()
	}
}

// isJSONContentType 检查 Content-Type 是否为 JSON 类型。
func isJSONContentType(ct string) bool {
	// 去除参数部分（如 "; charset=utf-8"）
	for i, c := range ct {
		if c == ';' {
			ct = ct[:i]
			break
		}
	}
	return ct == "application/json"
}
