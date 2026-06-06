package main

import (
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lcylpzls/ginx"
)

//go:embed all:dist
var spaAssets embed.FS

//go:embed all:public
var publicAssets embed.FS

func main() {
	// 提取 SPA 构建目录（去掉 dist/ 前缀）
	distFS, err := fs.Sub(spaAssets, "dist")
	if err != nil {
		panic(err)
	}

	// 提取 public 目录（去掉 public/ 前缀）
	publicFS, err := fs.Sub(publicAssets, "public")
	if err != nil {
		panic(err)
	}

	err = ginx.NewServer(ginx.Config{
		TLSCertFile:     "/etc/ssl/certs/server.crt",
		TLSKeyFile:      "/etc/ssl/private/server.key",
		ShutdownTimeout: 30 * time.Second,
		RequestTimeout:  30 * time.Second,
	}).
		UseHttp2Listen(":8443").
		// 1. 从本地目录提供静态文件
		ServeStaticDir("/public", "./public").
		// 2. 从 embed.FS 提供 SPA 构建产物
		ServeStaticFS("/", http.FS(distFS)).
		// 3. 启用 SPA 回退模式：未匹配的 GET 请求返回 index.html
		EnableSPA(http.FS(distFS), "index.html").
		// 4. 同时提供嵌入式 public 目录（另一个前缀）
		ServeStaticFS("/embed-public", http.FS(publicFS)).
		// 5. API 路由
		RegisterRoute(ginx.Route{
			Method: "GET",
			Path:   "/api/hello",
			Handler: func(c *gin.Context) {
				c.JSON(200, ginx.StandardizedResponse{
					Code:      ginx.CodeSuccess,
					Msg:       "Hello from ginx API!",
					RequestID: c.GetString("requestId"),
					Timestamp: time.Now().UnixMilli(),
				})
			},
		}).
		Start()

	if err != nil {
		panic(err)
	}
}
