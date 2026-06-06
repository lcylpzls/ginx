// Package ginx 提供基于 Gin 的工业级解耦 HTTPS Server 组件库。
//
// ginx 将 Gin 引擎完全封装在内部，对外暴露接口与配置，实现框架与业务的彻底解耦。
// 三方调用方无需直接依赖 Gin，仅需导入本包即可构建生产级 HTTPS 服务。
//
// v0.2.0 起强制 TLS，支持 HTTP/2、HTTP/3 (QUIC) 和 Unix Socket 多通道同时监听。
package ginx

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"
)

// Config 定义 ginx Server 的全部配置项。
//
// 配置由调用方显式构造后传入 NewServer，ginx 不提供 DefaultConfig()。
// 所有校验在 Validate() 中集中进行，失败返回简体中文错误消息。
type Config struct {
	// TLSCertFile TLS 证书文件路径（PEM 格式），必填。
	// 文件必须存在且为普通文件（不可为目录）。
	TLSCertFile string

	// TLSKeyFile TLS 私钥文件路径（PEM 格式），必填。
	// 文件必须存在且为普通文件（不可为目录）。
	TLSKeyFile string

	// ReadTimeout HTTP 读取超时时间。
	ReadTimeout time.Duration

	// WriteTimeout HTTP 写入超时时间。
	WriteTimeout time.Duration

	// IdleTimeout HTTP 空闲连接超时时间。
	IdleTimeout time.Duration

	// RequestTimeout 单个请求的超时时间，由 Timeout 中间件使用。
	RequestTimeout time.Duration

	// ShutdownTimeout 优雅关闭的最大等待时间。
	ShutdownTimeout time.Duration

	// MaxHeaderBytes 请求头的最大字节数。
	MaxHeaderBytes int

	// HealthPath 健康检查端点路径，默认为 "/health"。
	HealthPath string

	// LogLevel 日志级别，可选 "debug"、"info"、"warn"、"error"，为空则默认 "info"。
	LogLevel string

	// LogSuccessReq 是否记录成功请求的日志。
	LogSuccessReq bool

	// CORSAllowedOrigins CORS 允许的来源列表。
	CORSAllowedOrigins []string

	// CORSAllowedMethods CORS 允许的 HTTP 方法列表。
	CORSAllowedMethods []string

	// CORSAllowedHeaders CORS 允许的请求头列表。
	CORSAllowedHeaders []string

	// CORSMaxAge CORS 预检请求的缓存时间。
	CORSMaxAge time.Duration

	// MiddlewareRequestID 是否启用 RequestID 中间件。
	MiddlewareRequestID bool

	// MiddlewareCORS 是否启用 CORS 中间件。
	MiddlewareCORS bool

	// MiddlewareTimeout 是否启用 Timeout 中间件。
	MiddlewareTimeout bool

	// MiddlewareRecovery 是否启用 Recovery 中间件。
	MiddlewareRecovery bool

	// MiddlewareValidation 是否启用 Validation 中间件。
	MiddlewareValidation bool
}

// isRegularFile 检查给定路径是否存在且为普通文件（非目录）。
func isRegularFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("文件不存在：%s", path)
		}
		return fmt.Errorf("无法访问文件：%s：%w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("路径是目录而非文件：%s", path)
	}
	return nil
}

// Validate 对 Config 进行完整性校验，失败返回简体中文错误消息。
//
// 校验规则：
//   - TLSCertFile 不能为空，文件必须存在且为普通文件
//   - TLSKeyFile 不能为空，文件必须存在且为普通文件
//   - 证书和私钥能够成功配对（tls.LoadX509KeyPair 成功）
//   - 所有超时参数不能为负数
//   - LogLevel 为空默认 "info"，否则必须为有效级别
//   - HealthPath 为空默认 "/health"
func (c *Config) Validate() error {
	// TLS 证书校验
	if c.TLSCertFile == "" {
		return fmt.Errorf("ginx：TLS 证书文件路径不能为空")
	}
	if err := isRegularFile(c.TLSCertFile); err != nil {
		return fmt.Errorf("ginx：TLS 证书%s", err.Error())
	}

	// TLS 私钥校验
	if c.TLSKeyFile == "" {
		return fmt.Errorf("ginx：TLS 私钥文件路径不能为空")
	}
	if err := isRegularFile(c.TLSKeyFile); err != nil {
		return fmt.Errorf("ginx：TLS 私钥%s", err.Error())
	}

	// 证书与私钥配对校验
	if _, err := tls.LoadX509KeyPair(c.TLSCertFile, c.TLSKeyFile); err != nil {
		return fmt.Errorf("ginx：TLS 证书加载失败：%w", err)
	}

	// 超时参数校验
	if c.ShutdownTimeout < 0 {
		return fmt.Errorf("ginx：关闭超时时间不能为负数")
	}
	if c.RequestTimeout < 0 {
		return fmt.Errorf("ginx：请求超时时间不能为负数")
	}
	if c.ReadTimeout < 0 {
		return fmt.Errorf("ginx：读取超时时间不能为负数")
	}
	if c.WriteTimeout < 0 {
		return fmt.Errorf("ginx：写入超时时间不能为负数")
	}
	if c.IdleTimeout < 0 {
		return fmt.Errorf("ginx：空闲超时时间不能为负数")
	}
	if c.MaxHeaderBytes < 0 {
		return fmt.Errorf("ginx：最大请求头字节数不能为负数")
	}

	// 日志级别校验
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("ginx：日志级别无效：%s，必须是 debug、info、warn 或 error", c.LogLevel)
	}

	// HealthPath 默认值
	if c.HealthPath == "" {
		c.HealthPath = "/health"
	}

	return nil
}
