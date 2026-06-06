package ginx

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

// GracefulShutdown 监听系统信号并执行优雅关闭。
//
// 收到 SIGINT 或 SIGTERM 后，调用 httpServer.Shutdown(ctx) 排空请求，
// 清理 Unix Socket 文件（如适用），最后调用 cleanup 函数。
// 返回 nil 表示正常关闭，返回 error 表示关闭超时或异常。
//
// 注意：v0.2.0 起，此函数仅作为公开 API 保留，Server.Start() 内部已内联信号处理。
// 调用方如需自定义关闭逻辑可使用此函数。
func GracefulShutdown(
	ctx context.Context,
	logger Logger,
	httpServer *http.Server,
	listener net.Listener,
	shutdownTimeout time.Duration,
	unixSocketPath string,
	cleanupFuncs []func(),
) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	return gracefulShutdown(ctx, logger, httpServer, listener, shutdownTimeout, unixSocketPath, cleanupFuncs, quit)
}

// gracefulShutdown 内部实现，接受外部传入的信号通道用于测试。
func gracefulShutdown(
	ctx context.Context,
	logger Logger,
	httpServer *http.Server,
	listener net.Listener,
	shutdownTimeout time.Duration,
	unixSocketPath string,
	cleanupFuncs []func(),
	quit <-chan os.Signal,
) error {
	select {
	case sig := <-quit:
		logger.Info(ctx, fmt.Sprintf("ginx：收到系统信号 %s，开始优雅关闭", sig.String()))

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error(ctx, "ginx：服务关闭失败", ErrorField(err))
		}

		if listener != nil {
			listener.Close()
		}

		// Unix Socket 文件清理
		if unixSocketPath != "" && runtime.GOOS != "windows" {
			if err := os.Remove(unixSocketPath); err != nil && !os.IsNotExist(err) {
				logger.Warn(ctx, "ginx：残留 Socket 文件清理失败", StringField("路径", unixSocketPath), ErrorField(err))
			}
		}

		// 执行清理函数
		for _, fn := range cleanupFuncs {
			if fn != nil {
				fn()
			}
		}

		logger.Info(ctx, "ginx：服务已优雅关闭")
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}
