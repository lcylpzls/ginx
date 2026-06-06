# ginx

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/lcylpzls/ginx.svg)](https://pkg.go.dev/github.com/lcylpzls/ginx)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Coverage](https://img.shields.io/badge/coverage-83.8%25-brightgreen.svg)]()

**基于 Gin 的工业级 HTTPS Server 组件库**

*一套 API，多通道 TLS 服务 —— HTTP/2、HTTP/3 (QUIC)、Unix Socket 同时监听*

</div>

---

## 📖 目录

- [设计理念](#-设计理念)
- [核心特性](#-核心特性)
- [快速开始](#-快速开始)
- [安装](#-安装)
- [架构概览](#-架构概览)
- [使用指南](#-使用指南)
  - [基础用法](#1-基础用法)
  - [链式 API 配置](#2-链式-api-配置)
  - [多通道监听](#3-多通道监听)
  - [路由注册](#4-路由注册)
  - [路由分组](#5-路由分组)
  - [中间件管理](#6-中间件管理)
  - [自定义 Logger](#7-自定义-logger)
  - [IP 限流](#8-ip-限流)
  - [健康检查](#9-健康检查)
  - [优雅关闭](#10-优雅关闭)
  - [静态文件服务与 SPA](#11-静态文件服务与-spa)
- [API 参考](#-api-参考)
- [配置参考](#-配置参考)
- [内置中间件](#-内置中间件)
- [响应规范](#-响应规范)
- [最佳实践](#-最佳实践)
- [版本历史](#-版本历史)
- [许可证](#-许可证)

---

## 🎯 设计理念

ginx 将 [Gin](https://github.com/gin-gonic/gin) 引擎完全封装在内部，对外暴露纯接口与配置结构体，实现 **框架与业务的彻底解耦**。

### 为什么选择 ginx？

| 场景 | 直接使用 Gin | 使用 ginx |
|------|-------------|----------|
| 更换 Web 框架 | 修改所有 Handler 签名 | 仅修改 ginx 内部实现 |
| 多通道监听 | 自行实现 TCP+TLS+QUIC+Unix | 一行链式调用 |
| 统一日志/错误格式 | 自行封装中间件 | 开箱即用 |
| 团队规范约束 | 依靠 Code Review | 编译器强制 |

### 编码铁律

1. **绝不 panic** —— 唯一 `recover()` 在 `middleware/recovery.go` 内
2. **绝不吞错** —— 所有 error 必须上报或累加
3. **不导出 Gin 类型** —— `HandlerFunc` 是唯一暴露的 Gin 别名
4. **Start() 后不可变** —— 链式配置在启动后调用仅记录 Warn 日志

---

## ✨ 核心特性

- 🔒 **强制 TLS** —— 所有通道必须配置证书，拒绝明文传输
- 🌐 **多通道监听** —— HTTP/2 (TLS over TCP)、HTTP/3 (QUIC over UDP)、Unix Domain Socket 可同时开启
- 🔗 **链式 API** —— 流畅的 Builder 模式，配置即文档
- 🧱 **内置中间件** —— Recovery、RequestID、Timeout、CORS、Validation、RateLimit 开箱即用
- 📊 **标准化响应** —— 统一的 JSON 响应格式 `{code, msg, data, requestId, timestamp}`
- 🩺 **健康检查** —— 自动注册 `/health` 端点，可通过配置自定义路径
- 🛡️ **IP 限流** —— 基于令牌桶的 per-IP 限流，支持白名单
- 🪵 **可插拔日志** —— 通过 `Logger` 接口接入任意日志库（Zap、Zerolog、Logrus 等）
- 🧹 **优雅关闭** —— 支持 SIGINT/SIGTERM 信号捕获与 `Stop()` 主动关闭
- 🪟 **跨平台** —— Linux / macOS / Windows（Unix Socket 需 Windows 10 build 17063+）
- 🧪 **高覆盖率** —— 83.8% 测试覆盖率 + `-race` 零告警

---

## 🚀 快速开始

### 30 秒运行第一个 ginx 服务

```go
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
        ShutdownTimeout: 30 * time.Second,
        RequestTimeout:  30 * time.Second,
    }).
        UseHttp2Listen(":8443").
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
        }).
        Start()

    if err != nil {
        panic(err)
    }
}
```

```bash
# 生成自签名证书（开发环境）
bash gen_cert.sh

# 运行
go run main.go

# 测试
curl -k https://localhost:8443/ping
# {"code":0,"msg":"pong","requestId":"...","timestamp":...}

curl -k https://localhost:8443/health
# {"code":0,"msg":"ok","data":{"status":"运行中","uptime":"30秒","started":"..."},...}
```

---

## 📦 安装

```bash
go get github.com/lcylpzls/ginx@latest
```

**依赖项：**

| 依赖 | 用途 |
|------|------|
| `github.com/gin-gonic/gin` | HTTP 引擎 |
| `github.com/quic-go/quic-go` | HTTP/3 (QUIC) 协议支持 |

要求 Go ≥ 1.21。

---

## 🏗️ 架构概览

```
┌─────────────────────────────────────────────────┐
│                    ginx.Server                    │
│  ┌─────────────────────────────────────────────┐│
│  │              Chain API (Builder)             ││
│  │  WithLogger → UseHttp2Listen → UseHttp3Listen││
│  │  → RegisterRoute → EnableRateLimit → Start  ││
│  └──────────────────┬──────────────────────────┘│
│                     │ Start()                     │
│         ┌───────────┼───────────┐                │
│         ▼           ▼           ▼                │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐         │
│  │ HTTP/2   │ │ HTTP/3   │ │  Unix    │         │
│  │ TLS:TCP  │ │ QUIC:UDP │ │  Socket  │         │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘         │
│       │            │            │                │
│       └────────────┼────────────┘                │
│                    ▼                             │
│         ┌─────────────────────┐                  │
│         │    gin.Engine        │                  │
│         │  ┌───────────────┐   │                  │
│         │  │  Middleware    │   │                  │
│         │  │  Chain         │   │                  │
│         │  │  Recovery      │   │                  │
│         │  │  RequestID     │   │                  │
│         │  │  Timeout       │   │                  │
│         │  │  CORS          │   │                  │
│         │  │  Validation    │   │                  │
│         │  │  RateLimit     │   │                  │
│         │  ├───────────────┤   │                  │
│         │  │  Routes        │   │                  │
│         │  │  /health       │   │                  │
│         │  └───────────────┘   │                  │
│         └─────────────────────┘                  │
└─────────────────────────────────────────────────┘
```

---

## 📚 使用指南

### 1. 基础用法

```go
package main

import (
    "time"
    "github.com/lcylpzls/ginx"
)

func main() {
    s := ginx.NewServer(ginx.Config{
        TLSCertFile:     "/path/to/cert.pem",
        TLSKeyFile:      "/path/to/key.pem",
        ShutdownTimeout: 30 * time.Second,
        RequestTimeout:  30 * time.Second,
    })

    s.UseHttp2Listen(":8443")

    if err := s.Start(); err != nil {
        panic(err)
    }
}
```

### 2. 链式 API 配置

所有配置方法均返回 `*Server`，支持流畅的链式调用：

```go
err := ginx.NewServer(cfg).
    WithLogger(myLogger).              // 注入自定义 Logger
    UseHttp2Listen(":8443").           // 启用 HTTP/2
    UseHttp3Listen(":8443").           // 启用 HTTP/3（同端口，TCP/UDP 不冲突）
    UseGlobalMiddleware(authMiddleware).// 全局中间件
    DisableMiddleware(ginx.MiddlewareValidation). // 禁用校验中间件
    OverrideMiddleware(ginx.MiddlewareRequestID, customRequestID). // 覆盖中间件
    RegisterRoute(ginx.Route{...}).    // 注册单条路由
    RegisterRoutes([]ginx.Route{...}). // 批量注册路由
    RegisterRouteGroup("/api", func(rg *ginx.RouteGroup) { // 路由分组
        rg.GET("/users", listUsers)
    }).
    EnableRateLimit(ginx.RateLimitOptions{ // IP 限流
        QPS:    100,
        Window: time.Second,
    }).
    Start()
```

> ⚠️ **重要**: 在 `Start()` 之后调用链式方法不会 panic，但会通过 Logger 输出 Warn 警告并忽略修改。

### 3. 多通道监听

ginx 支持同时启用多种监听通道：

```go
s := ginx.NewServer(cfg).
    UseHttp2Listen("0.0.0.0:443").            // HTTPS — 对外服务
    UseHttp3Listen("0.0.0.0:443").            // HTTP/3 QUIC — 同端口
    UseUnixSocketListen("/run/app.sock", 0660) // Unix Socket — 本地通信
```

| 方法 | 协议 | 传输层 | 典型场景 |
|------|------|--------|---------|
| `UseHttp2Listen(addr)` | HTTP/1.1 + HTTP/2 | TLS over TCP | 对外 API 服务 |
| `UseHttp3Listen(addr)` | HTTP/3 | QUIC over UDP | 移动端/弱网优化 |
| `UseUnixSocketListen(path, perm)` | HTTP/1.1 | Unix Domain Socket | 本地 Nginx 反向代理 |

### 4. 路由注册

#### 单条路由

```go
s.RegisterRoute(ginx.Route{
    Method: "GET",
    Path:   "/api/users/:id",
    Handler: func(c *gin.Context) {
        userID := c.Param("id")
        c.JSON(200, ginx.StandardizedResponse{
            Code:      ginx.CodeSuccess,
            Msg:       "ok",
            Data:      map[string]string{"id": userID},
            RequestID: c.GetString("requestId"),
            Timestamp: time.Now().UnixMilli(),
        })
    },
    Middleware: []ginx.HandlerFunc{authMiddleware}, // 路由专属中间件（可选）
})
```

#### 批量注册

```go
s.RegisterRoutes([]ginx.Route{
    {Method: "GET",  Path: "/api/users",    Handler: listUsers},
    {Method: "POST", Path: "/api/users",    Handler: createUser},
    {Method: "PUT",  Path: "/api/users/:id", Handler: updateUser},
})
```

### 5. 路由分组

```go
s.RegisterRouteGroup("/api/v2", func(rg *ginx.RouteGroup) {
    // 分组级中间件
    rg.Use(authMiddleware, rateLimitMiddleware)

    // 子路由
    rg.GET("/products", listProducts)
    rg.POST("/products", createProduct)

    // 嵌套分组
    admin := rg.Group("/admin")
    admin.GET("/stats", getStats)
})
```

### 6. 中间件管理

#### 内置中间件开关

通过 `Config` 控制：

```go
cfg := ginx.Config{
    MiddlewareRecovery:   true,  // 默认开启
    MiddlewareRequestID:  true,  // 默认开启
    MiddlewareTimeout:    true,  // 默认开启
    MiddlewareCORS:       true,  // 默认开启
    MiddlewareValidation: true,  // 默认开启
}
```

#### 运行时控制

```go
// 禁用
s.DisableMiddleware(ginx.MiddlewareValidation, ginx.MiddlewareCORS)

// 重新启用（RateLimit 除外）
s.EnableMiddleware(ginx.MiddlewareValidation)

// 覆盖内置实现
s.OverrideMiddleware(ginx.MiddlewareRequestID, myCustomRequestID)

// 追加全局中间件（追加到链末尾）
s.UseGlobalMiddleware(loggingMiddleware, metricsMiddleware)
```

#### 中间件执行顺序

```
RequestID → CORS → Recovery → Timeout → Validation → RateLimit → Handler
                                                                    ↑
                                              全局中间件（UseGlobalMiddleware）
```

### 7. 自定义 Logger

实现 `ginx.Logger` 接口即可接入任意日志库：

```go
// 使用 Zap 的示例
type ZapLogger struct {
    logger *zap.Logger
}

func (l *ZapLogger) Debug(ctx context.Context, msg string, fields ...ginx.Field) {
    l.logger.Debug(msg, toZapFields(fields)...)
}
func (l *ZapLogger) Info(ctx context.Context, msg string, fields ...ginx.Field) {
    l.logger.Info(msg, toZapFields(fields)...)
}
func (l *ZapLogger) Warn(ctx context.Context, msg string, fields ...ginx.Field) {
    l.logger.Warn(msg, toZapFields(fields)...)
}
func (l *ZapLogger) Error(ctx context.Context, msg string, fields ...ginx.Field) {
    l.logger.Error(msg, toZapFields(fields)...)
}
func (l *ZapLogger) Fatal(ctx context.Context, msg string, fields ...ginx.Field) {
    l.logger.Fatal(msg, toZapFields(fields)...)
}

// 注入
s.WithLogger(&ZapLogger{logger: zapLogger})
```

**日志字段构建器：**

```go
ginx.StringField("path", "/api/users")
ginx.IntField("status", 200)
ginx.DurationField("latency", elapsed)
ginx.ErrorField(err)
ginx.AnyField("body", data)
```

### 8. IP 限流

基于令牌桶算法的 per-IP 限流，支持 CIDR 白名单：

```go
s.EnableRateLimit(ginx.RateLimitOptions{
    QPS:     100,              // 每 IP 每秒允许的请求数
    Window:  time.Second,      // 限流窗口
    Whitelist: []string{       // 白名单（IP 或 CIDR）
        "10.0.0.0/8",
        "127.0.0.1",
    },
    CleanupInterval: 5 * time.Minute, // 过期桶清理间隔
})
```

**客户端 IP 提取优先级：** `X-Forwarded-For` > `X-Real-IP` > `RemoteAddr`

### 9. 健康检查

ginx 自动注册健康检查端点，默认路径 `/health`：

```go
// 自定义路径
cfg := ginx.Config{
    HealthPath: "/api/healthz",
}

// 若自定义路径与业务路由冲突，业务路由优先
s.RegisterRoute(ginx.Route{
    Method: "GET",
    Path:   "/api/healthz",  // 覆盖内置健康检查
    Handler: customHealthHandler,
})
```

**响应示例：**

```json
{
    "code": 0,
    "msg": "ok",
    "data": {
        "status": "运行中",
        "uptime": "2小时30分钟",
        "started": "2026-06-06T10:00:00+08:00"
    },
    "requestId": "",
    "timestamp": 1749200400000
}
```

### 10. 优雅关闭

#### 主动关闭

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := s.Stop(ctx); err != nil {
    log.Printf("关闭服务出错：%v", err)
}
```

#### 信号关闭（Start 内置）

`Start()` 内部已注册 SIGINT/SIGTERM 信号处理，收到信号后自动执行优雅关闭：

```go
// Start() 内部流程：
// 1. 创建 Listener
// 2. 启动 HTTP Server goroutine
// 3. 注册 signal.Notify(quit, SIGINT, SIGTERM)
// 4. 阻塞等待信号或 ctx.Done()
// 5. 收到信号 → Shutdown 所有 HTTP Server
// 6. Close 所有 Listener
// 7. 清理 Unix Socket 文件
// 8. 执行 cleanupFuncs
```

#### 独立使用 GracefulShutdown

```go
ginx.GracefulShutdown(
    ctx,
    logger,
    httpServer,
    listener,
    30*time.Second,    // shutdownTimeout
    "/run/app.sock",   // unixSocketPath（空字符串表示不清理）
    cleanupFuncs,
)
```

### 11. 静态文件服务与 SPA

ginx 支持从本地目录或 Go embed.FS 提供静态文件，并内置 SPA 回退模式（未匹配的 GET 请求自动返回 index.html）。

#### 本地目录

```go
s := ginx.NewServer(cfg).
    UseHttp2Listen(":443").
    ServeStaticDir("/assets", "./public").  // 等价于 gin.Static()
    RegisterRoute(ginx.Route{
        Method: "GET", Path: "/api/hello",
        Handler: helloHandler,
    })
```

#### 嵌入式文件系统（embed.FS）

```go
//go:embed all:frontend/dist
var distFS embed.FS

func main() {
    s := ginx.NewServer(cfg).
        UseHttp2Listen(":443").
        ServeStaticFS("/", http.FS(distFS)).  // 等价于 gin.StaticFS()
        RegisterRoute(...).
        Start()
}
```

#### SPA 模式

当使用 Vue Router / React Router 等客户端路由时，浏览器直接访问 `/dashboard` 在服务端没有对应文件，需要回退到 `index.html`：

```go
//go:embed all:frontend/dist
var spaAssets embed.FS

func main() {
    distFS, _ := fs.Sub(spaAssets, "frontend/dist")

    err := ginx.NewServer(cfg).
        UseHttp2Listen(":443").
        ServeStaticFS("/", http.FS(distFS)).        // 提供所有静态文件
        EnableSPA(http.FS(distFS), "index.html").     // SPA 回退
        RegisterRoute(ginx.Route{
            Method: "GET", Path: "/api/hello",
            Handler: helloHandler,
        }).
        Start()
}
```

> **SPA 回退规则**：仅对未匹配的 GET/HEAD 请求返回 index.html。POST/PUT/DELETE 等非 GET/HEAD 请求仍返回标准 JSON 404/405 响应。API 路由优先于 SPA 回退。

#### 多静态前缀

同时提供多个静态目录或文件系统：

```go
s.ServeStaticDir("/docs", "./public/docs").
  ServeStaticFS("/admin", http.FS(adminFS)).
  ServeStaticFS("/", http.FS(distFS)).
  EnableSPA(http.FS(distFS), "index.html")
```

> **注意**：`ServeStaticDir` 使用 `http.Dir` 实现，在 Linux 下与 SPA 的 `EnableSPA` 无冲突。各前缀通过 gin 的 radix tree 独立路由，互不干扰。

---

## 📋 API 参考

### 核心函数

| 函数 | 说明 |
|------|------|
| `NewServer(cfg Config) *Server` | 创建 Server 实例 |
| `GracefulShutdown(ctx, logger, srv, ln, timeout, sockPath, cleanups) error` | 独立优雅关闭函数 |

### Server 方法

#### 生命周期

| 方法 | 说明 |
|------|------|
| `Start() error` | 启动服务（阻塞，直到收到信号或错误） |
| `Stop(ctx context.Context) error` | 优雅关闭（sync.Once 保护，可重复调用） |
| `ListenerAddr() string` | 返回首个 Listener 的监听地址（线程安全） |

#### 链式配置

| 方法 | 说明 |
|------|------|
| `WithLogger(l Logger) *Server` | 注入自定义 Logger 实现 |
| `UseHttp2Listen(addr string) *Server` | 启用 HTTP/2 TLS 监听 |
| `UseHttp3Listen(addr string) *Server` | 启用 HTTP/3 QUIC 监听 |
| `UseUnixSocketListen(path string, perm os.FileMode) *Server` | 启用 Unix Domain Socket 监听 |
| `RegisterRoute(r Route) *Server` | 注册单条路由 |
| `RegisterRoutes(routes []Route) *Server` | 批量注册路由 |
| `RegisterRouteGroup(prefix string, fn func(*RouteGroup)) *Server` | 注册路由分组 |
| `UseGlobalMiddleware(mw ...HandlerFunc) *Server` | 追加全局中间件 |
| `OverrideMiddleware(mt MiddlewareType, mw HandlerFunc) *Server` | 覆盖内置中间件实现 |
| `DisableMiddleware(mt ...MiddlewareType) *Server` | 禁用内置中间件 |
| `EnableMiddleware(mt ...MiddlewareType) *Server` | 启用内置中间件（RateLimit 需用 EnableRateLimit） |
| `EnableRateLimit(opts RateLimitOptions) *Server` | 启用 IP 令牌桶限流 |
| `DisableRateLimit() *Server` | 禁用 IP 限流 |
| `ServeStaticDir(prefix, root string) *Server` | 从本地目录提供静态文件 |
| `ServeStaticFS(prefix string, fs http.FileSystem) *Server` | 从 http.FileSystem 提供静态文件（支持 embed.FS） |
| `EnableSPA(fs http.FileSystem, indexPath string) *Server` | 启用 SPA 回退模式 |

### 数据类型

#### `Config`

全部配置项的结构体，详见 [配置参考](#-配置参考)。

#### `Route`

```go
type Route struct {
    Method     string          // HTTP 方法
    Path       string          // 路由路径
    Handler    HandlerFunc     // 处理器
    Middleware []HandlerFunc   // 路由专属中间件（可选）
}
```

#### `RouteGroup`

路由分组，支持嵌套和分组级中间件。

```go
rg.GET(path, handler, mw...)     // GET
rg.POST(path, handler, mw...)    // POST
rg.PUT(path, handler, mw...)     // PUT
rg.DELETE(path, handler, mw...)  // DELETE
rg.PATCH(path, handler, mw...)   // PATCH
rg.Use(mw...)                    // 中间件
rg.Group(relativePath) *RouteGroup // 子分组
```

#### `StandardizedResponse`

统一的 JSON 响应体：

```go
type StandardizedResponse struct {
    Code      int    `json:"code"`
    Msg       string `json:"msg"`
    Data      any    `json:"data,omitempty"`
    RequestID string `json:"requestId"`
    Timestamp int64  `json:"timestamp"`
}
```

#### `HandlerFunc`

```go
type HandlerFunc = gin.HandlerFunc  // ginx 暴露的唯一 Gin 类型别名
```

#### `MiddlewareType`

```go
type MiddlewareType string

const (
    MiddlewareRequestID  MiddlewareType = "request_id"
    MiddlewareCORS       MiddlewareType = "cors"
    MiddlewareTimeout    MiddlewareType = "timeout"
    MiddlewareRecovery   MiddlewareType = "recovery"
    MiddlewareValidation MiddlewareType = "validation"
    MiddlewareRateLimit  MiddlewareType = "rate_limit"
)
```

#### `RateLimitOptions`

```go
type RateLimitOptions struct {
    QPS             int           // 每 IP 每秒请求数（必填，> 0）
    Window          time.Duration // 限流窗口（必填，> 0）
    Whitelist       []string      // IP/CIDR 白名单（可选）
    CleanupInterval time.Duration // 过期桶清理间隔（可选，默认 5 分钟）
}
```

### 日志接口

```go
type Logger interface {
    Debug(ctx context.Context, msg string, fields ...Field)
    Info(ctx context.Context, msg string, fields ...Field)
    Warn(ctx context.Context, msg string, fields ...Field)
    Error(ctx context.Context, msg string, fields ...Field)
    Fatal(ctx context.Context, msg string, fields ...Field)
}

type Field struct {
    Key   string
    Value any
}

// 字段构建器
func StringField(key, val string) Field
func IntField(key string, val int) Field
func DurationField(key string, val time.Duration) Field
func ErrorField(err error) Field
func AnyField(key string, val any) Field
```

`NoopLogger` 是默认的空实现，所有方法不执行任何操作。

---

## ⚙️ 配置参考

```go
type Config struct {
    // === TLS 证书（必填）===
    TLSCertFile string   // TLS 证书文件路径（PEM 格式）
    TLSKeyFile  string   // TLS 私钥文件路径（PEM 格式）

    // === 超时 ===
    ReadTimeout     time.Duration // HTTP 读取超时
    WriteTimeout    time.Duration // HTTP 写入超时
    IdleTimeout     time.Duration // 空闲连接超时
    RequestTimeout  time.Duration // 请求处理超时（Timeout 中间件使用）
    ShutdownTimeout time.Duration // 优雅关闭最大等待时间

    // === HTTP ===
    MaxHeaderBytes int    // 请求头最大字节数
    HealthPath     string // 健康检查路径（默认 "/health"）
    LogLevel       string // 日志级别：debug/info/warn/error（默认 "info"）
    LogSuccessReq  bool   // 是否记录成功请求

    // === CORS ===
    CORSAllowedOrigins []string      // 允许的来源
    CORSAllowedMethods []string      // 允许的 HTTP 方法
    CORSAllowedHeaders []string      // 允许的请求头
    CORSMaxAge         time.Duration // 预检请求缓存时间

    // === 中间件开关 ===
    MiddlewareRequestID  bool // RequestID 中间件
    MiddlewareCORS       bool // CORS 中间件
    MiddlewareTimeout    bool // Timeout 中间件
    MiddlewareRecovery   bool // Recovery 中间件
    MiddlewareValidation bool // Validation 中间件
}
```

### 状态码常量

| 常量 | 值 | 说明 |
|------|---|------|
| `CodeSuccess` | 0 | 请求成功 |
| `CodeBadRequest` | 400 | 请求参数校验失败 |
| `CodeNotFound` | 404 | 请求的资源不存在 |
| `CodeMethodNotAllowed` | 405 | 不支持的请求方法 |
| `CodeTooManyRequests` | 429 | 请求频率超限 |
| `CodeInternalError` | 500 | 服务器内部错误 |
| `CodeServiceUnavailable` | 503 | 服务暂时不可用（超时） |

---

## 🧩 内置中间件

| 中间件 | 类型标识 | 说明 | 默认 |
|--------|---------|------|------|
| **Recovery** | `MiddlewareRecovery` | Panic 捕获，返回 500 + 堆栈日志 | 启用 |
| **RequestID** | `MiddlewareRequestID` | 生成 UUID v4 格式请求 ID，设置 `X-Request-ID` 响应头 | 启用 |
| **Timeout** | `MiddlewareTimeout` | 请求超时控制，超时返回 503，使用自定义 ResponseWriter 无 data race | 启用 |
| **CORS** | `MiddlewareCORS` | 跨域处理，通过 Config 配置规则 | 启用 |
| **Validation** | `MiddlewareValidation` | POST/PUT/PATCH 请求体 JSON 格式校验，跳过 GET/HEAD/OPTIONS | 启用 |
| **RateLimit** | `MiddlewareRateLimit` | IP 令牌桶限流，需通过 `EnableRateLimit()` 显式开启 | 关闭 |

---

## 📊 响应规范

所有响应（包括 404/405/429/500/503）均使用 `StandardizedResponse` 格式：

### 正常响应

```json
{
    "code": 0,
    "msg": "ok",
    "data": {"id": "123", "name": "Alice"},
    "requestId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "timestamp": 1749200400000
}
```

### 异常响应对照

| HTTP 状态码 | `code` | `msg` | 触发条件 |
|------------|--------|-------|---------|
| 400 | 400 | 请求参数校验失败 | Validation 中间件拦截 |
| 404 | 404 | 请求的资源不存在 | 路径无匹配路由 |
| 405 | 405 | 不支持的请求方法 | 方法不匹配（需 `HandleMethodNotAllowed=true`） |
| 429 | 429 | — | RateLimit 超限 |
| 500 | 500 | 服务器内部错误 | Panic 被 Recovery 捕获 |
| 503 | 503 | 请求处理超时 | Timeout 中间件超时 |

---

## 💡 最佳实践

### 1. 证书管理

```bash
# 开发/测试环境：使用项目自带脚本生成自签名证书
bash gen_cert.sh

# 生产环境：使用 Let's Encrypt 或企业 CA 签发的证书
# 确保证书文件权限为 600
chmod 600 /etc/ssl/private/server.key
```

### 2. 在 Handler 中获取 RequestID

```go
func myHandler(c *gin.Context) {
    requestID := c.GetString("requestId")
    // 用于日志追踪和响应
}
```

### 3. 使用响应常量

```go
// ✅ 推荐：使用 ginx 提供的状态码常量
c.JSON(http.StatusBadRequest, ginx.StandardizedResponse{
    Code:      ginx.CodeBadRequest,
    Msg:       "参数校验失败：" + err.Error(),
    RequestID: c.GetString("requestId"),
    Timestamp: time.Now().UnixMilli(),
})

// ❌ 避免：硬编码数字
c.JSON(400, ginx.StandardizedResponse{Code: 400, ...})
```

### 4. 生产环境配置参考

```go
cfg := ginx.Config{
    TLSCertFile:          "/etc/ssl/certs/prod.crt",
    TLSKeyFile:           "/etc/ssl/private/prod.key",
    ReadTimeout:          10 * time.Second,
    WriteTimeout:         30 * time.Second,
    IdleTimeout:          120 * time.Second,
    RequestTimeout:       30 * time.Second,
    ShutdownTimeout:      30 * time.Second,
    MaxHeaderBytes:       1 << 20, // 1 MB
    HealthPath:           "/healthz",
    LogLevel:             "info",
    LogSuccessReq:        false,
    MiddlewareRecovery:   true,
    MiddlewareRequestID:  true,
    MiddlewareTimeout:    true,
    MiddlewareCORS:       true,
    MiddlewareValidation: true,
    CORSAllowedOrigins:   []string{"https://example.com"},
    CORSAllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    CORSAllowedHeaders:   []string{"Content-Type", "Authorization"},
    CORSMaxAge:           12 * time.Hour,
}
```

### 5. 框架解耦

```go
// ✅ ginx 使用 HandlerFunc = gin.HandlerFunc 别名
// 当未来切换底层框架时，只需修改 ginx 内部的类型映射
// 调用方代码无需改动（只要不直接 import gin）
func myHandler(c *gin.Context) {
    c.JSON(200, ginx.StandardizedResponse{...})
}
```

---

## 📝 版本历史

| 版本 | 日期 | 里程碑 |
|------|------|--------|
| **v0.9.0** | 2026-06 | Timeout Data Race 修复、panic → Warn 日志、83.8% 覆盖率 |
| v0.4.0 | 2026-06 | Stop() 吞错修复、链式方法并发控制、Handler 私有化 |
| v0.3.0 | — | Windows AF_UNIX 兼容性支持 |
| v0.2.0 | — | 多通道 TLS/HTTP3/Unix Socket Listener |
| v0.1.0 | — | 初始版本：Gin 封装 + 中间件管理 |

---

## 📄 许可证

[MIT](LICENSE)

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给一个 Star！**

</div>
