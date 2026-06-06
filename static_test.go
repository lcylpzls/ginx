package ginx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// createTestFS 在临时目录中创建文件并返回 http.FileSystem。
// files 是文件名到内容的映射。
func createTestFS(t *testing.T, files map[string]string) http.FileSystem {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		fullPath := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("创建目录失败：%v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("创建文件失败：%v", err)
		}
	}
	// 使用 os.DirFS + http.FS 而非 http.Dir，避免 http.Dir 在 Linux
	// 下与 gin.FileFromFS 组合时因路径解析产生的 301 重定向问题。
	return http.FS(os.DirFS(dir))
}

// ======================== spaNoRoute 单元测试 ========================

func TestSpaNoRoute_GET_ReturnsIndexHTML(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := createTestFS(t, map[string]string{
		"index.html": "<html><body>SPA</body></html>",
	})

	r := gin.New()
	r.RedirectTrailingSlash = false
	r.NoRoute(spaNoRoute(fs, "index.html"))

	req := httptest.NewRequest(http.MethodGet, "/random-path", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, w.Code)
	}
	if body := w.Body.String(); body != "<html><body>SPA</body></html>" {
		t.Errorf("期望 body 为 index.html 内容，实际 %q", body)
	}
}

func TestSpaNoRoute_HEAD_ReturnsIndexHTMLHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := createTestFS(t, map[string]string{
		"index.html": "<html><body>SPA</body></html>",
	})

	r := gin.New()
	r.RedirectTrailingSlash = false
	r.NoRoute(spaNoRoute(fs, "index.html"))

	req := httptest.NewRequest(http.MethodHead, "/random-path", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, w.Code)
	}
}

func TestSpaNoRoute_POST_ReturnsJSON404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := createTestFS(t, map[string]string{
		"index.html": "<html><body>SPA</body></html>",
	})

	r := gin.New()
	r.RedirectTrailingSlash = false
	r.NoRoute(spaNoRoute(fs, "index.html"))

	req := httptest.NewRequest(http.MethodPost, "/random-path", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusNotFound, w.Code)
	}

	var resp StandardizedResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON 解析失败：%v", err)
	}
	if resp.Code != http.StatusNotFound {
		t.Errorf("期望 code %d，实际 %d", http.StatusNotFound, resp.Code)
	}
	if resp.Msg != "请求的资源不存在" {
		t.Errorf("期望 msg '请求的资源不存在'，实际 %q", resp.Msg)
	}
}

func TestSpaNoRoute_PUT_ReturnsJSON404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := createTestFS(t, map[string]string{
		"index.html": "<html></html>",
	})

	r := gin.New()
	r.RedirectTrailingSlash = false
	r.NoRoute(spaNoRoute(fs, "index.html"))

	req := httptest.NewRequest(http.MethodPut, "/random-path", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望 PUT 请求状态码 %d，实际 %d", http.StatusNotFound, w.Code)
	}
}

func TestSpaNoRoute_DELETE_ReturnsJSON404(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := createTestFS(t, map[string]string{
		"index.html": "<html></html>",
	})

	r := gin.New()
	r.RedirectTrailingSlash = false
	r.NoRoute(spaNoRoute(fs, "index.html"))

	req := httptest.NewRequest(http.MethodDelete, "/random-path", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望 DELETE 请求状态码 %d，实际 %d", http.StatusNotFound, w.Code)
	}
}

// ======================== ServeStaticFS 单元测试 ========================

func TestServeStaticFS_ServesFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := createTestFS(t, map[string]string{
		"hello.txt": "Hello, World!",
		"data.json": `{"key": "value"}`,
	})

	r := gin.New()
	r.StaticFS("/static", fs)

	req := httptest.NewRequest(http.MethodGet, "/static/hello.txt", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, w.Code)
	}
	if body := w.Body.String(); body != "Hello, World!" {
		t.Errorf("期望 body 'Hello, World!'，实际 %q", body)
	}
}

func TestServeStaticFS_JSONFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := createTestFS(t, map[string]string{
		"data.json": `{"key": "value"}`,
	})

	r := gin.New()
	r.StaticFS("/", fs)

	req := httptest.NewRequest(http.MethodGet, "/data.json", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, w.Code)
	}
	if contentType := w.Header().Get("Content-Type"); contentType == "" {
		t.Error("期望有 Content-Type 响应头")
	}
}

func TestServeStaticFS_NestedPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := createTestFS(t, map[string]string{
		"js/app.js":    "console.log('app');",
		"css/style.css": "body { margin: 0; }",
	})

	r := gin.New()
	r.StaticFS("/assets", fs)

	// 嵌套路径
	req := httptest.NewRequest(http.MethodGet, "/assets/js/app.js", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, w.Code)
	}
	if body := w.Body.String(); body != "console.log('app');" {
		t.Errorf("期望 body 为 js 内容，实际 %q", body)
	}
}

// ======================== 集成测试 ========================

func TestServer_ServeStaticFS_Integration(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	fs := createTestFS(t, map[string]string{
		"hello.txt": "Hello from embed!",
	})

	cfg := Config{
		TLSCertFile:          certFile,
		TLSKeyFile:           keyFile,
		ShutdownTimeout:      time.Second,
		MiddlewareRecovery:   false,
		MiddlewareRequestID:  false,
		MiddlewareTimeout:    false,
		MiddlewareCORS:       false,
		MiddlewareValidation: false,
	}

	s := NewServer(cfg).
		UseHttp2Listen("127.0.0.1:0").
		ServeStaticFS("/static", fs)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	time.Sleep(200 * time.Millisecond)

	tr := &http.Transport{}
	client := &http.Client{Transport: tr, Timeout: 2 * time.Second}
	resp, err := client.Get("https://" + s.ListenerAddr() + "/static/hello.txt")
	if err != nil {
		t.Fatalf("请求失败：%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Stop(ctx)
	<-errCh
}

func TestServer_ServeStaticDir_Integration(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "info.txt"), []byte("dir content"), 0o644); err != nil {
		t.Fatalf("创建文件失败：%v", err)
	}

	cfg := Config{
		TLSCertFile:          certFile,
		TLSKeyFile:           keyFile,
		ShutdownTimeout:      time.Second,
		MiddlewareRecovery:   false,
		MiddlewareRequestID:  false,
		MiddlewareTimeout:    false,
		MiddlewareCORS:       false,
		MiddlewareValidation: false,
	}

	s := NewServer(cfg).
		UseHttp2Listen("127.0.0.1:0").
		ServeStaticDir("/files", dir)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	time.Sleep(200 * time.Millisecond)

	tr := &http.Transport{}
	client := &http.Client{Transport: tr, Timeout: 2 * time.Second}
	resp, err := client.Get("https://" + s.ListenerAddr() + "/files/info.txt")
	if err != nil {
		t.Fatalf("请求失败：%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusOK, resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Stop(ctx)
	<-errCh
}

func TestServer_SPA_Fallback_GET(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	fs := createTestFS(t, map[string]string{
		"index.html": "<!DOCTYPE html><html><body>SPA App</body></html>",
		"app.js":     "console.log('app');",
	})

	cfg := Config{
		TLSCertFile:          certFile,
		TLSKeyFile:           keyFile,
		ShutdownTimeout:      time.Second,
		MiddlewareRecovery:   false,
		MiddlewareRequestID:  false,
		MiddlewareTimeout:    false,
		MiddlewareCORS:       false,
		MiddlewareValidation: false,
	}

	s := NewServer(cfg).
		UseHttp2Listen("127.0.0.1:0").
		ServeStaticFS("/", fs).
		EnableSPA(fs, "index.html")

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	time.Sleep(200 * time.Millisecond)

	tr := &http.Transport{}
	client := &http.Client{Transport: tr, Timeout: 2 * time.Second}

	// 已知文件应该被 ServeStaticFS 匹配
	resp, err := client.Get("https://" + s.ListenerAddr() + "/app.js")
	if err != nil {
		t.Fatalf("请求 /app.js 失败：%v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /app.js 期望状态码 %d，实际 %d", http.StatusOK, resp.StatusCode)
	}

	// 未知路径应该回退到 index.html（SPA）
	resp2, err := client.Get("https://" + s.ListenerAddr() + "/dashboard/settings")
	if err != nil {
		t.Fatalf("请求 /dashboard/settings 失败：%v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("GET /dashboard/settings 期望状态码 %d（SPA 回退），实际 %d", http.StatusOK, resp2.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Stop(ctx)
	<-errCh
}

func TestServer_SPA_POST_Returns404(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	fs := createTestFS(t, map[string]string{
		"index.html": "<html></html>",
	})

	cfg := Config{
		TLSCertFile:          certFile,
		TLSKeyFile:           keyFile,
		ShutdownTimeout:      time.Second,
		MiddlewareRecovery:   false,
		MiddlewareRequestID:  false,
		MiddlewareTimeout:    false,
		MiddlewareCORS:       false,
		MiddlewareValidation: false,
	}

	s := NewServer(cfg).
		UseHttp2Listen("127.0.0.1:0").
		EnableSPA(fs, "index.html")

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	time.Sleep(200 * time.Millisecond)

	tr := &http.Transport{}
	client := &http.Client{Transport: tr, Timeout: 2 * time.Second}
	resp, err := client.Post("https://"+s.ListenerAddr()+"/random-path", "text/plain", nil)
	if err != nil {
		t.Fatalf("POST 请求失败：%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST 期望状态码 %d，实际 %d", http.StatusNotFound, resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Stop(ctx)
	<-errCh
}

func TestServer_WithoutSPA_NoRouteReturnsJSON404(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	cfg := Config{
		TLSCertFile:          certFile,
		TLSKeyFile:           keyFile,
		ShutdownTimeout:      time.Second,
		MiddlewareRecovery:   false,
		MiddlewareRequestID:  false,
		MiddlewareTimeout:    false,
		MiddlewareCORS:       false,
		MiddlewareValidation: false,
	}

	s := NewServer(cfg).
		UseHttp2Listen("127.0.0.1:0")

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	time.Sleep(200 * time.Millisecond)

	tr := &http.Transport{}
	client := &http.Client{Transport: tr, Timeout: 2 * time.Second}
	resp, err := client.Get("https://" + s.ListenerAddr() + "/nonexistent")
	if err != nil {
		t.Fatalf("请求失败：%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("期望状态码 %d，实际 %d", http.StatusNotFound, resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Stop(ctx)
	<-errCh
}

func TestServer_MultipleStaticPrefixes(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	fs1 := createTestFS(t, map[string]string{
		"a.txt": "file A",
	})
	fs2 := createTestFS(t, map[string]string{
		"b.txt": "file B",
	})

	cfg := Config{
		TLSCertFile:          certFile,
		TLSKeyFile:           keyFile,
		ShutdownTimeout:      time.Second,
		MiddlewareRecovery:   false,
		MiddlewareRequestID:  false,
		MiddlewareTimeout:    false,
		MiddlewareCORS:       false,
		MiddlewareValidation: false,
	}

	s := NewServer(cfg).
		UseHttp2Listen("127.0.0.1:0").
		ServeStaticFS("/assets", fs1).
		ServeStaticFS("/static", fs2)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	time.Sleep(200 * time.Millisecond)

	tr := &http.Transport{}
	client := &http.Client{Transport: tr, Timeout: 2 * time.Second}

	resp1, _ := client.Get("https://" + s.ListenerAddr() + "/assets/a.txt")
	if resp1 != nil {
		resp1.Body.Close()
		if resp1.StatusCode != http.StatusOK {
			t.Errorf("GET /assets/a.txt 期望 %d，实际 %d", http.StatusOK, resp1.StatusCode)
		}
	}

	resp2, _ := client.Get("https://" + s.ListenerAddr() + "/static/b.txt")
	if resp2 != nil {
		resp2.Body.Close()
		if resp2.StatusCode != http.StatusOK {
			t.Errorf("GET /static/b.txt 期望 %d，实际 %d", http.StatusOK, resp2.StatusCode)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Stop(ctx)
	<-errCh
}
