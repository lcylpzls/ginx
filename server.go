package ginx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lcylpzls/ginx/middleware"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

// routeGroupEntry 记录路由分组的配置，在 Start() 时展开注册。
type routeGroupEntry struct {
	prefix string
	fn     func(*RouteGroup)
}

// Server 是 ginx 的核心类型，封装 Gin 引擎并提供工业级 HTTPS Server 能力。
//
// 通过链式 API 进行配置，调用 Start() 启动服务，Stop(ctx) 优雅关闭。
// v0.2.0 起支持 HTTP/2、HTTP/3 (QUIC) 和 Unix Socket 多通道同时监听。
type Server struct {
	config        Config
	logger        Logger
	routes        []Route
	routeGroups   []routeGroupEntry
	staticEntries []staticEntry
	spa           *spaConfig
	mwManager     *middleware.Manager
	rateLimiter   *middleware.RateLimiter
	engine        *gin.Engine
	startTime     time.Time
	started       bool
	mu            sync.Mutex
	shutdownOnce  sync.Once
	cleanupFuncs  []func()

	// 多 Listener 管理
	listeners   []net.Listener
	listenersMu sync.Mutex
	httpServers []*http.Server // 所有 http.Server 实例（用于 Shutdown）

	// 监听配置
	http2Enabled   bool
	http2Addr      string
	http3Enabled   bool
	http3Addr      string
	unixEnabled    bool
	unixSocketPath string
	unixSocketPerm os.FileMode
}

// NewServer 创建一个新的 ginx Server 实例。
//
// cfg 由调用方显式构造，ginx 不提供默认配置。
func NewServer(cfg Config) *Server {
	return &Server{
		config:    cfg,
		logger:    NoopLogger{},
		mwManager: middleware.NewManager(),
	}
}

// WithLogger 注入自定义 Logger 实现。
func (s *Server) WithLogger(l Logger) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	s.logger = l
	return s
}

// UseGlobalMiddleware 追加外部全局中间件。
func (s *Server) UseGlobalMiddleware(mw ...HandlerFunc) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	s.mwManager.Append(mw...)
	return s
}

// OverrideMiddleware 使用自定义 Handler 覆盖指定类型的内置中间件。
func (s *Server) OverrideMiddleware(mt MiddlewareType, mw HandlerFunc) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	s.mwManager.Override(string(mt), mw)
	return s
}

// DisableMiddleware 禁用指定类型的内置中间件。
func (s *Server) DisableMiddleware(mt ...MiddlewareType) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	keys := make([]string, len(mt))
	for i, t := range mt {
		keys[i] = string(t)
	}
	s.mwManager.Disable(keys...)
	return s
}

// EnableMiddleware 重新启用指定类型的内置中间件（RateLimit 除外）。
func (s *Server) EnableMiddleware(mt ...MiddlewareType) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	keys := make([]string, len(mt))
	for i, t := range mt {
		keys[i] = string(t)
	}
	s.mwManager.Enable(keys...)
	return s
}

// RegisterRoute 注册单条路由。
func (s *Server) RegisterRoute(r Route) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	s.routes = append(s.routes, r)
	return s
}

// RegisterRoutes 批量注册路由。
func (s *Server) RegisterRoutes(routes []Route) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	s.routes = append(s.routes, routes...)
	return s
}

// RegisterRouteGroup 注册路由分组。
//
// fn 在 Start() 时被调用以展开分组内路由。
func (s *Server) RegisterRouteGroup(prefix string, fn func(*RouteGroup)) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	s.routeGroups = append(s.routeGroups, routeGroupEntry{prefix: prefix, fn: fn})
	return s
}

// UseHttp2Listen 启用 HTTP/2 TLS 监听（含 HTTP/1.1 兼容）。
//
// addr 格式如 ":443" 或 "0.0.0.0:8888"。
func (s *Server) UseHttp2Listen(addr string) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	s.http2Enabled = true
	s.http2Addr = addr
	return s
}

// UseHttp3Listen 启用 HTTP/3 QUIC 监听。
//
// addr 独立指定，与 HTTP/2 无关。格式如 ":443" 或 "127.0.0.1:9900"。
func (s *Server) UseHttp3Listen(addr string) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	s.http3Enabled = true
	s.http3Addr = addr
	return s
}

// UseUnixSocketListen 启用 Unix Socket 监听。
//
// perm 为 Socket 文件权限，为 0 则默认 0660。
func (s *Server) UseUnixSocketListen(path string, perm os.FileMode) *Server {
	// Windows 版本检测（非 Windows 平台编译为空操作）
	requireWindowsBuild()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	s.unixEnabled = true
	s.unixSocketPath = path
	if perm == 0 {
		perm = 0660
	}
	s.unixSocketPerm = perm
	return s
}

// EnableRateLimit 启用 IP 限流中间件。
func (s *Server) EnableRateLimit(opts RateLimitOptions) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}

	if opts.QPS <= 0 || opts.Window <= 0 {
		return s
	}

	rl := middleware.NewRateLimiter(opts.QPS, opts.Window, opts.Whitelist)
	s.rateLimiter = rl

	s.mwManager.EnableRateLimit(middleware.RateLimit(rl))

	cleanupInterval := opts.CleanupInterval
	if cleanupInterval <= 0 {
		cleanupInterval = 5 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cleanupFuncs = append(s.cleanupFuncs, cancel)

	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rl.Cleanup(cleanupInterval)
			}
		}
	}()

	return s
}

// DisableRateLimit 禁用 IP 限流中间件。
func (s *Server) DisableRateLimit() *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	s.mwManager.DisableRateLimit()
	s.rateLimiter = nil
	return s
}

// addListener 将 Listener 注册到 Server 的内部管理中。
func (s *Server) addListener(ln net.Listener) {
	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()
	s.listeners = append(s.listeners, ln)
}

// addHTTPServer 将 http.Server 注册到内部管理中。
func (s *Server) addHTTPServer(srv *http.Server) {
	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()
	s.httpServers = append(s.httpServers, srv)
}

// ListenerAddr 返回第一个 Listener 的监听地址，线程安全。
//
// 当使用 port 0 动态分配端口时，可通过此方法获取实际监听端口。
// 若尚未创建任何 Listener，返回空字符串。
func (s *Server) ListenerAddr() string {
	s.listenersMu.Lock()
	defer s.listenersMu.Unlock()
	if len(s.listeners) > 0 {
		return s.listeners[0].Addr().String()
	}
	return ""
}

// Start 启动 HTTPS 服务。
//
// 执行配置校验、中间件加载、路由注册、各通道监听器创建和启动，
// 调用后阻塞直到收到关闭信号或发生错误。
func (s *Server) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("ginx：服务已启动，不允许重复启动")
	}
	s.started = true
	s.mu.Unlock()

	// 0. 校验至少一种监听方式
	if !s.http2Enabled && !s.http3Enabled && !s.unixEnabled {
		return fmt.Errorf("ginx：至少需要启用一种监听方式（HTTP/2、HTTP/3 或 Unix Socket）")
	}

	// 1. Config.Validate()
	if err := s.config.Validate(); err != nil {
		return fmt.Errorf("ginx：配置校验失败：%w", err)
	}

	// 2. 记录 startTime
	s.startTime = time.Now()

	// 3. gin.New() 创建引擎
	s.engine = gin.New()

	// 4. 注册内置中间件到 Manager
	s.registerBuiltinMiddleware()

	// 5. mwManager.Build() → engine.Use()
	ctx := context.Background()
	chain := s.mwManager.Build(ctx)
	for _, mw := range chain {
		s.engine.Use(mw)
	}

	// 6. 注册 Route
	for _, route := range s.routes {
		handlers := append(route.Middleware, route.Handler)
		s.engine.Handle(route.Method, route.Path, handlers...)
	}

	// 7. 注册 RouteGroup
	for _, entry := range s.routeGroups {
		rg := &RouteGroup{prefix: entry.prefix}
		entry.fn(rg)
		for _, route := range rg.flatten() {
			handlers := append(route.Middleware, route.Handler)
			s.engine.Handle(route.Method, route.Path, handlers...)
		}
	}

	// 8. 注册 /health
	healthPath := s.config.HealthPath
	healthRegistered := false
	for _, route := range s.routes {
		if route.Path == healthPath {
			healthRegistered = true
			break
		}
	}
	if !healthRegistered {
		for _, entry := range s.routeGroups {
			rg := &RouteGroup{prefix: entry.prefix}
			entry.fn(rg)
			for _, route := range rg.flatten() {
				if route.Path == healthPath {
					healthRegistered = true
					break
				}
			}
			if healthRegistered {
				break
			}
		}
	}
	if !healthRegistered {
		s.engine.GET(healthPath, healthHandler(s.startTime))
	}

	// 9. 注册静态文件服务
	for _, entry := range s.staticEntries {
		s.engine.StaticFS(entry.prefix, entry.fs)
	}

	// 10. NoRoute / NoMethod 兜底
	s.engine.HandleMethodNotAllowed = true
	if s.spa != nil {
		s.engine.NoRoute(spaNoRoute(s.spa.fs, s.spa.indexPath))
	} else {
		s.engine.NoRoute(noRouteHandler)
	}
	s.engine.NoMethod(noMethodHandler)

	// 10. 创建并启动各通道 Listener
	var wg sync.WaitGroup

	// HTTP/2 (TLS over TCP)
	if s.http2Enabled {
		ln, err := createTLSListener(s.http2Addr, s.config.TLSCertFile, s.config.TLSKeyFile)
		if err != nil {
			return err
		}
		s.addListener(ln)
		http2Server := &http.Server{
			Handler:        s.engine,
			ReadTimeout:    s.config.ReadTimeout,
			WriteTimeout:   s.config.WriteTimeout,
			IdleTimeout:    s.config.IdleTimeout,
			MaxHeaderBytes: s.config.MaxHeaderBytes,
		}
		s.addHTTPServer(http2Server)
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Info(ctx, "ginx：HTTP/2 服务已启动", StringField("地址", s.http2Addr))
			if err := http2Server.Serve(ln); err != nil && err != http.ErrServerClosed {
				s.logger.Error(ctx, "ginx：HTTP/2 服务异常退出", ErrorField(err))
			}
		}()
	}

	// HTTP/3 (QUIC over UDP)
	if s.http3Enabled {
		qln, err := createQUICListener(s.http3Addr, s.config.TLSCertFile, s.config.TLSKeyFile)
		if err != nil {
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Info(ctx, "ginx：HTTP/3 (QUIC) 服务已启动", StringField("地址", s.http3Addr))
			if err := serveHTTP3(ctx, s.engine, qln); err != nil {
				s.logger.Error(ctx, "ginx：HTTP/3 服务异常退出", ErrorField(err))
			}
		}()
	}

	// Unix Socket
	if s.unixEnabled {
		uln, err := createUnixListener(s.unixSocketPath, s.unixSocketPerm)
		if err != nil {
			return err
		}
		s.addListener(uln)
		unixServer := &http.Server{
			Handler:        s.engine,
			ReadTimeout:    s.config.ReadTimeout,
			WriteTimeout:   s.config.WriteTimeout,
			IdleTimeout:    s.config.IdleTimeout,
			MaxHeaderBytes: s.config.MaxHeaderBytes,
		}
		s.addHTTPServer(unixServer)
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.logger.Info(ctx, "ginx：Unix Socket 服务已启动", StringField("路径", s.unixSocketPath))
			if err := unixServer.Serve(uln); err != nil && err != http.ErrServerClosed {
				s.logger.Error(ctx, "ginx：Unix Socket 服务异常退出", ErrorField(err))
			}
		}()
	}

	// 11. 信号处理 goroutine
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(quit)

		select {
		case sig := <-quit:
			s.logger.Info(ctx, fmt.Sprintf("ginx：收到系统信号 %s，开始优雅关闭", sig.String()))
			shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
			defer cancel()

			// 关闭所有 HTTP Server（快照读取，避免 data race）
			s.listenersMu.Lock()
			httpServers := make([]*http.Server, len(s.httpServers))
			copy(httpServers, s.httpServers)
			s.listenersMu.Unlock()
			for _, srv := range httpServers {
				if err := srv.Shutdown(shutdownCtx); err != nil {
					s.logger.Error(ctx, "ginx：服务关闭失败", ErrorField(err))
				}
			}

			// 关闭所有 Listener
			s.listenersMu.Lock()
			for _, ln := range s.listeners {
				ln.Close()
			}
			s.listenersMu.Unlock()

			// Unix Socket 文件清理
			if s.unixEnabled && runtime.GOOS != "windows" {
				if err := os.Remove(s.unixSocketPath); err != nil && !os.IsNotExist(err) {
					s.logger.Warn(ctx, "ginx：残留 Socket 文件清理失败", StringField("路径", s.unixSocketPath), ErrorField(err))
				}
			}

			// 执行清理函数
			for _, fn := range s.cleanupFuncs {
				if fn != nil {
					fn()
				}
			}
			s.logger.Info(ctx, "ginx：服务已优雅关闭")
		case <-ctx.Done():
		}
	}()

	// 12. 等待所有 Listener 退出
	wg.Wait()
	return nil
}

// Stop 优雅关闭 HTTPS 服务。
//
// 使用 sync.Once 保护，重复调用安全。
// 关闭所有 HTTP Server 和 Listener。
func (s *Server) Stop(ctx context.Context) error {
	var err error
	s.shutdownOnce.Do(func() {
		// 关闭所有 HTTP Server（快照读取，避免 data race）
		s.listenersMu.Lock()
		httpServers := make([]*http.Server, len(s.httpServers))
		copy(httpServers, s.httpServers)
		s.listenersMu.Unlock()
		for _, srv := range httpServers {
			shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
			defer cancel()
			if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
				err = errors.Join(err, shutdownErr)
			}
		}

		// 关闭所有 Listener
		s.listenersMu.Lock()
		for _, ln := range s.listeners {
			if closeErr := ln.Close(); closeErr != nil {
				err = errors.Join(err, closeErr)
			}
		}
		s.listenersMu.Unlock()

		// Unix Socket 文件清理
		if s.unixEnabled && runtime.GOOS != "windows" {
			if removeErr := os.Remove(s.unixSocketPath); removeErr != nil && !os.IsNotExist(removeErr) {
				s.logger.Warn(ctx, "ginx：残留 Socket 文件清理失败",
					StringField("路径", s.unixSocketPath),
					ErrorField(removeErr))
				err = errors.Join(err, removeErr)
			}
		}

		// 执行清理函数
		for _, fn := range s.cleanupFuncs {
			if fn != nil {
				fn()
			}
		}
	})
	return err
}

// registerBuiltinMiddleware 根据 Config 的中间件开关，向 Manager 注册对应的中间件。
func (s *Server) registerBuiltinMiddleware() {
	s.mwManager.RegisterBuiltin("recovery", middleware.Recovery())
	if !s.config.MiddlewareRecovery {
		s.mwManager.Disable("recovery")
	}

	s.mwManager.RegisterBuiltin("request_id", middleware.RequestID())
	if !s.config.MiddlewareRequestID {
		s.mwManager.Disable("request_id")
	}

	timeoutMiddleware := middleware.Timeout(s.config.RequestTimeout)
	s.mwManager.RegisterBuiltin("timeout", timeoutMiddleware)
	if !s.config.MiddlewareTimeout {
		s.mwManager.Disable("timeout")
	}

	corsCfg := middleware.CORSConfig{
		AllowedOrigins: s.config.CORSAllowedOrigins,
		AllowedMethods: s.config.CORSAllowedMethods,
		AllowedHeaders: s.config.CORSAllowedHeaders,
		MaxAge:         int(s.config.CORSMaxAge.Seconds()),
	}
	if len(corsCfg.AllowedOrigins) == 0 {
		corsCfg = middleware.DefaultCORSConfig()
	}
	s.mwManager.RegisterBuiltin("cors", middleware.CORS(corsCfg))
	if !s.config.MiddlewareCORS {
		s.mwManager.Disable("cors")
	}

	s.mwManager.RegisterBuiltin("validation", middleware.Validation())
	if !s.config.MiddlewareValidation {
		s.mwManager.Disable("validation")
	}
}

// serveHTTP3 在指定的 QUIC Listener 上运行 HTTP/3 服务。
//
// 循环接受 QUIC 连接，为每个连接在独立 goroutine 中调用 http3.ServeQUICConn。
func serveHTTP3(ctx context.Context, handler http.Handler, qln *quic.Listener) error {
	h3s := &http3.Server{Handler: handler}
	for {
		conn, err := qln.Accept(ctx)
		if err != nil {
			return err
		}
		go h3s.ServeQUICConn(conn)
	}
}
