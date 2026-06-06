package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lcylpzls/ginx"
)

func main() {
	s := ginx.NewServer(ginx.Config{
		TLSCertFile:     "/etc/ssl/certs/server.crt",
		TLSKeyFile:      "/etc/ssl/private/server.key",
		ShutdownTimeout: 30 * time.Second,
		RequestTimeout:  30 * time.Second,
	})

	s.UseHttp2Listen(":8443").
		RegisterRoute(ginx.Route{
			Method: "GET",
			Path:   "/ping",
			Handler: func(c *gin.Context) {
				c.JSON(200, ginx.StandardizedResponse{
					Code:      ginx.CodeSuccess,
					Msg:       "pong",
					RequestID: c.GetString("requestId"),
					Timestamp: time.Now().UnixMilli(),
				})
			},
		})

	if err := s.Start(); err != nil {
		panic(err)
	}
}
