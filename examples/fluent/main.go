package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lcylpzls/ginx"
)

func main() {
	err := ginx.NewServer(ginx.Config{
		TLSCertFile:     "/etc/ssl/certs/server.crt",
		TLSKeyFile:      "/etc/ssl/private/server.key",
		RequestTimeout:  30 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}).
		UseHttp2Listen("0.0.0.0:8888").
		UseHttp3Listen("127.0.0.1:9900").
		UseUnixSocketListen("/var/run/app.sock", 0660).
		DisableMiddleware(ginx.MiddlewareValidation).
		RegisterRoute(ginx.Route{
			Method:  "GET",
			Path:    "/api/users/:id",
			Handler: getUser,
		}).
		RegisterRouteGroup("/api/v2", func(rg *ginx.RouteGroup) {
			rg.GET("/products", listProducts)
			rg.POST("/products", createProduct)
		}).
		EnableRateLimit(ginx.RateLimitOptions{
			QPS:    200,
			Window: time.Second,
		}).
		Start()

	if err != nil {
		panic(err)
	}
}

func getUser(c *gin.Context) {
	c.JSON(200, ginx.StandardizedResponse{
		Code:      ginx.CodeSuccess,
		Msg:       "ok",
		Data:      map[string]string{"id": c.Param("id"), "name": "Alice"},
		RequestID: c.GetString("requestId"),
		Timestamp: time.Now().UnixMilli(),
	})
}

func listProducts(c *gin.Context) {
	c.JSON(200, ginx.StandardizedResponse{
		Code:      ginx.CodeSuccess,
		Msg:       "ok",
		Data:      []string{"Product A", "Product B"},
		RequestID: c.GetString("requestId"),
		Timestamp: time.Now().UnixMilli(),
	})
}

func createProduct(c *gin.Context) {
	c.JSON(201, ginx.StandardizedResponse{
		Code:      ginx.CodeSuccess,
		Msg:       "创建成功",
		RequestID: c.GetString("requestId"),
		Timestamp: time.Now().UnixMilli(),
	})
}
