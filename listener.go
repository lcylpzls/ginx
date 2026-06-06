package ginx

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/quic-go/quic-go"
)

// createTLSListener 创建 TLS over TCP 监听器，支持 HTTP/2 和 HTTP/1.1。
func createTLSListener(addr, certFile, keyFile string) (net.Listener, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("ginx：TLS 证书加载失败：%w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"h2", "http/1.1"},
	}

	ln, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("ginx：TLS 监听失败，地址 %s：%w", addr, err)
	}
	return ln, nil
}

// createQUICListener 创建 QUIC over UDP 监听器，用于 HTTP/3。
func createQUICListener(addr, certFile, keyFile string) (*quic.Listener, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("ginx：QUIC 证书加载失败：%w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3"},
	}

	ln, err := quic.ListenAddr(addr, tlsConfig, &quic.Config{
		MaxIdleTimeout: 30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("ginx：QUIC 监听失败，地址 %s：%w", addr, err)
	}
	return ln, nil
}

// createUnixListener 创建 Unix Socket 监听器。
//
// 非 Windows 下先清理残留 Socket 文件，再创建监听并设置权限。
func createUnixListener(path string, perm os.FileMode) (net.Listener, error) {
	// 清理残留 Socket 文件
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("ginx：Unix Socket 残留文件清理失败，路径 %s：%w", path, err)
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("ginx：Unix Socket 监听失败，路径 %s：%w", path, err)
	}

	// 设置 Socket 文件权限
	if err := os.Chmod(path, perm); err != nil {
		ln.Close()
		return nil, fmt.Errorf("ginx：Unix Socket 权限设置失败，路径 %s：%w", path, err)
	}

	return ln, nil
}

// resolveUnixSocketPath 返回规范的 Socket 路径。
//
// Windows 10 build 1803+ 原生支持 AF_UNIX，无需路径转换。
func resolveUnixSocketPath(address string) string {
	return address
}
