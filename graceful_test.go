package ginx

import (
	"context"
	"net"
	"net/http"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestGracefulShutdown_SignalReceived(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("创建 Listener 失败：%v", err)
	}
	defer ln.Close()

	httpServer := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	go httpServer.Serve(ln)

	logger := NoopLogger{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		quit <- syscall.SIGTERM
	}()

	cleanupCalled := false
	cleanup := []func(){func() { cleanupCalled = true }}

	err = gracefulShutdown(ctx, logger, httpServer, ln, 5*time.Second, "", cleanup, quit)
	if err != nil {
		t.Errorf("期望 nil error，实际 %v", err)
	}
	if !cleanupCalled {
		t.Error("期望 cleanup 函数被调用")
	}
}

func TestGracefulShutdown_ContextCancelled(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("创建 Listener 失败：%v", err)
	}
	defer ln.Close()

	httpServer := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	go httpServer.Serve(ln)

	logger := NoopLogger{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	quit := make(chan os.Signal)
	err = gracefulShutdown(ctx, logger, httpServer, ln, 5*time.Second, "", nil, quit)
	if err == nil {
		t.Fatal("期望返回 context.Canceled 错误，实际为 nil")
	}
}

func TestGracefulShutdown_NilCleanup(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("创建 Listener 失败：%v", err)
	}
	defer ln.Close()

	httpServer := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	go httpServer.Serve(ln)

	logger := NoopLogger{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		quit <- syscall.SIGTERM
	}()

	cleanup := []func(){nil, func() {}, nil}
	err = gracefulShutdown(ctx, logger, httpServer, ln, time.Second, "", cleanup, quit)
	if err != nil {
		t.Errorf("期望 nil error，实际 %v", err)
	}
}

func TestGracefulShutdown_NilListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("创建 Listener 失败：%v", err)
	}

	httpServer := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	go httpServer.Serve(ln)

	logger := NoopLogger{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		quit <- syscall.SIGTERM
	}()

	err = gracefulShutdown(ctx, logger, httpServer, nil, time.Second, "", nil, quit)
	if err != nil {
		t.Errorf("期望 nil error，实际 %v", err)
	}
	ln.Close()
}

func TestGracefulShutdown_UnixSocketCleanup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix Socket 测试仅在非 Windows 系统运行")
	}

	tmpDir := t.TempDir()
	sockPath := tmpDir + "/test.sock"

	ln, err := createUnixListener(sockPath, 0660)
	if err != nil {
		t.Fatalf("createUnixListener 失败：%v", err)
	}

	httpServer := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}

	go httpServer.Serve(ln)

	logger := NoopLogger{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		quit <- syscall.SIGTERM
	}()

	err = gracefulShutdown(ctx, logger, httpServer, ln, time.Second, sockPath, nil, quit)
	if err != nil {
		t.Errorf("期望 nil error，实际 %v", err)
	}

	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Error("期望 Socket 文件已被清理")
	}
}
