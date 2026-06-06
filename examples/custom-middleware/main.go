package main

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lcylpzlz/ginx"
)

// CustomLogger 演示自定义 Logger 实现。
type CustomLogger struct{}

func (l CustomLogger) Debug(_ context.Context, msg string, _ ...ginx.Field) {}
func (l CustomLogger) Info(_ context.Context, msg string, _ ...ginx.Field)  {}
func (l CustomLogger) Warn(_ context.Context, msg string, _ ...ginx.Field)  {}
func (l CustomLogger) Error(_ context.Context, msg string, _ ...ginx.Field) {}
func (l CustomLogger) Fatal(_ context.Context, msg string, _ ...ginx.Field) {}

func main() {
	ginx.NewServer(ginx.Config{
		TLSCertFile:     "/etc/ssl/certs/server.crt",
		TLSKeyFile:      "/etc/ssl/private/server.key",
		ShutdownTimeout: 30 * time.Second,
		RequestTimeout:  30 * time.Second,
	}).
		WithLogger(CustomLogger{}).
		UseHttp2Listen(":8443").
		UseHttp3Listen(":8443"). // 与 HTTP/2 同端口也可行（TCP/UDP 不冲突）
		OverrideMiddleware(ginx.MiddlewareRequestID, func(c *gin.Context) {
			c.Set("requestId", "custom-prefix-"+time.Now().Format("20060102150405"))
			c.Header("X-Request-ID", c.GetString("requestId"))
			c.Next()
		}).
		DisableMiddleware(ginx.MiddlewareValidation).
		UseGlobalMiddleware(func(c *gin.Context) {
			c.Header("X-Powered-By", "ginx")
			c.Next()
		}).
		RegisterRoute(ginx.Route{
			Method: "GET",
			Path:   "/hello",
			Handler: func(c *gin.Context) {
				c.JSON(200, ginx.StandardizedResponse{
					Code:      ginx.CodeSuccess,
					Msg:       "Hello, ginx!",
					RequestID: c.GetString("requestId"),
					Timestamp: time.Now().UnixMilli(),
				})
			},
		}).
		Start()
}
