package ginx

import (
	"os"
	"runtime"
	"testing"
)

func TestCreateTLSListener(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	ln, err := createTLSListener("127.0.0.1:0", certFile, keyFile)
	if err != nil {
		t.Fatalf("createTLSListener 失败：%v", err)
	}
	defer ln.Close()

	if ln == nil {
		t.Fatal("期望非 nil Listener")
	}
}

func TestCreateTLSListener_InvalidCert(t *testing.T) {
	_, err := createTLSListener("127.0.0.1:0", "/nonexistent/cert.pem", "/nonexistent/key.pem")
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
}

func TestCreateTLSListener_BindError(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	_, err := createTLSListener("256.256.256.256:0", certFile, keyFile)
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
}

func TestCreateQUICListener(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	qln, err := createQUICListener("127.0.0.1:0", certFile, keyFile)
	if err != nil {
		t.Fatalf("createQUICListener 失败：%v", err)
	}
	defer qln.Close()

	if qln == nil {
		t.Fatal("期望非 nil QUIC Listener")
	}
}

func TestCreateUnixListener(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix Socket 测试仅在非 Windows 系统运行")
	}

	tmpDir := t.TempDir()
	sockPath := tmpDir + "/test.sock"

	ln, err := createUnixListener(sockPath, 0660)
	if err != nil {
		t.Fatalf("createUnixListener 失败：%v", err)
	}
	defer ln.Close()

	if _, err := os.Stat(sockPath); os.IsNotExist(err) {
		t.Error("期望 Socket 文件被创建")
	}
}

func TestCreateUnixListener_CleanupStale(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix Socket 测试仅在非 Windows 系统运行")
	}

	tmpDir := t.TempDir()
	sockPath := tmpDir + "/test.sock"

	ln1, err := createUnixListener(sockPath, 0660)
	if err != nil {
		t.Fatalf("第一次 createUnixListener 失败：%v", err)
	}
	ln1.Close()

	ln2, err := createUnixListener(sockPath, 0660)
	if err != nil {
		t.Fatalf("第二次 createUnixListener 失败：%v", err)
	}
	ln2.Close()
}

func TestCreateUnixListener_WindowsSupported(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅在 Windows 系统运行")
	}

	// Windows 10 build 1803+ 原生支持 AF_UNIX
	// AF_UNIX 路径有长度限制（约 108 字节），必须使用短路径
	sockPath := os.TempDir() + "\\ginx-test.sock"
	os.Remove(sockPath) // 清理可能残留的旧文件

	ln, err := createUnixListener(sockPath, 0660)
	if err != nil {
		t.Fatalf("createUnixListener 在 Windows 上应成功，实际错误：%v", err)
	}
	ln.Close()
	os.Remove(sockPath)
}

func TestResolveUnixSocketPath(t *testing.T) {
	path := resolveUnixSocketPath("/var/run/app.sock")
	if path != "/var/run/app.sock" {
		t.Errorf("期望原路径 '/var/run/app.sock'，实际 %q", path)
	}
}
