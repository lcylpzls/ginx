package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// timeoutWriter 包装 gin.ResponseWriter，在超时后丢弃写入。
//
// 不引入额外 goroutine，避免 *gin.Context 的 data race 和 sync.Pool 污染。
type timeoutWriter struct {
	gin.ResponseWriter
	ctx         context.Context
	timedOut    bool
	wroteHeader bool
}

// WriteHeader 写入 HTTP 状态码。
// 若 Context 已超时，丢弃写入并标记 timedOut。
func (w *timeoutWriter) WriteHeader(code int) {
	select {
	case <-w.ctx.Done():
		w.timedOut = true
	default:
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(code)
	}
}

// Write 写入 HTTP 响应体。
// 若 Context 已超时，丢弃写入并标记 timedOut。
func (w *timeoutWriter) Write(b []byte) (int, error) {
	select {
	case <-w.ctx.Done():
		w.timedOut = true
		return len(b), nil
	default:
		return w.ResponseWriter.Write(b)
	}
}

// WriteString 写入字符串响应体。
func (w *timeoutWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// Timeout 返回一个请求超时中间件。
//
// 向请求注入带超时的 Context，下游 Handler 应响应 ctx.Done() 以提前退出。
// 超时后 Handler 的写入被丢弃，并返回 503 响应。
// 若 timeout <= 0，则中间件不执行任何操作，直接放行。
//
// 与旧版不同：不再将 c.Next() 放入 goroutine，避免 *gin.Context 的
// data race 和 sync.Pool 污染问题。
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if timeout <= 0 {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		origWriter := c.Writer
		tw := &timeoutWriter{
			ResponseWriter: origWriter,
			ctx:            ctx,
		}
		c.Writer = tw

		c.Next()

		// 恢复原始 Writer
		c.Writer = origWriter

		// 超时且 handler 未成功写入响应头 → 返回 503
		if tw.timedOut && !tw.wroteHeader {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code":      503,
				"msg":       "请求处理超时",
				"data":      nil,
				"requestId": c.GetString("requestId"),
				"timestamp": time.Now().UnixMilli(),
			})
		}
	}
}
