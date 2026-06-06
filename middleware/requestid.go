package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID 返回一个请求 ID 生成中间件。
//
// 优先使用请求头 X-Request-ID 的值，若不存在则生成 UUID v4。
// 生成的 ID 写入 gin.Context 的 "requestId" 键和响应头 X-Request-ID。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("requestId", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}
