package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

// Recovery 返回一个 Panic 捕获中间件。
//
// 这是组件库中唯一调用 recover() 的位置。
// 当 handler 中发生 panic 时，Recovery 捕获并记录调用栈，
// 向客户端返回标准化 500 响应。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()

				// 记录调用栈（日志由 Server 注入的 Logger 输出）
				// 此处无法直接获取 Logger，记录到 Context 的 error 中
				c.Set("recoveryError", fmt.Sprintf("ginx：请求处理发生 panic：%v", r))
				c.Set("recoveryStack", string(stack))

				// 返回标准化 500 响应
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":      500,
					"msg":       "服务器内部错误",
					"data":      nil,
					"requestId": c.GetString("requestId"),
					"timestamp": time.Now().UnixMilli(),
				})
			}
		}()
		c.Next()
	}
}
