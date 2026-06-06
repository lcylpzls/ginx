package ginx

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// generateTestCert 生成临时自签名证书和私钥，返回文件路径。
func generateTestCert(t *testing.T) (certFile, keyFile string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成 RSA 密钥失败：%v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"ginx Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("创建证书失败：%v", err)
	}

	tmpDir := t.TempDir()
	certFile = filepath.Join(tmpDir, "cert.pem")
	keyFile = filepath.Join(tmpDir, "key.pem")

	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("创建证书文件失败：%v", err)
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("创建私钥文件失败：%v", err)
	}
	defer keyOut.Close()
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return certFile, keyFile
}

func TestConfig_Validate_Valid(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	c := &Config{
		TLSCertFile:     certFile,
		TLSKeyFile:      keyFile,
		ShutdownTimeout: 30 * time.Second,
		RequestTimeout:  30 * time.Second,
		LogLevel:        "info",
		HealthPath:      "/health",
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("期望校验通过，实际返回错误：%v", err)
	}
}

func TestConfig_Validate_EmptyCert(t *testing.T) {
	c := &Config{
		TLSCertFile:     "",
		TLSKeyFile:      "/some/key.pem",
		ShutdownTimeout: time.Second,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if !strings.Contains(err.Error(), "TLS 证书文件路径不能为空") {
		t.Errorf("错误消息应包含 'TLS 证书文件路径不能为空'，实际 %q", err.Error())
	}
}

func TestConfig_Validate_EmptyKey(t *testing.T) {
	certFile, _ := generateTestCert(t)

	c := &Config{
		TLSCertFile:     certFile,
		TLSKeyFile:      "",
		ShutdownTimeout: time.Second,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if !strings.Contains(err.Error(), "TLS 私钥文件路径不能为空") {
		t.Errorf("错误消息应包含 'TLS 私钥文件路径不能为空'，实际 %q", err.Error())
	}
}

func TestConfig_Validate_CertFileNotExist(t *testing.T) {
	c := &Config{
		TLSCertFile:     "/nonexistent/cert.pem",
		TLSKeyFile:      "/nonexistent/key.pem",
		ShutdownTimeout: time.Second,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if !strings.Contains(err.Error(), "文件不存在") {
		t.Errorf("错误消息应包含 '文件不存在'，实际 %q", err.Error())
	}
}

func TestConfig_Validate_CertIsDir(t *testing.T) {
	certFile, keyFile := generateTestCert(t)
	tmpDir := t.TempDir()

	c := &Config{
		TLSCertFile:     tmpDir, // 目录而非文件
		TLSKeyFile:      keyFile,
		ShutdownTimeout: time.Second,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if !strings.Contains(err.Error(), "路径是目录而非文件") {
		t.Errorf("错误消息应包含 '路径是目录而非文件'，实际 %q", err.Error())
	}
	_ = certFile
}

func TestConfig_Validate_KeyIsDir(t *testing.T) {
	certFile, _ := generateTestCert(t)
	tmpDir := t.TempDir()

	c := &Config{
		TLSCertFile:     certFile,
		TLSKeyFile:      tmpDir, // 目录而非文件
		ShutdownTimeout: time.Second,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if !strings.Contains(err.Error(), "路径是目录而非文件") {
		t.Errorf("错误消息应包含 '路径是目录而非文件'，实际 %q", err.Error())
	}
}

func TestConfig_Validate_CertMismatch(t *testing.T) {
	certFile, _ := generateTestCert(t)
	// 生成另一个 key 来制造不匹配
	_, keyFile2 := generateTestCert(t)

	c := &Config{
		TLSCertFile:     certFile,
		TLSKeyFile:      keyFile2, // 不匹配的私钥
		ShutdownTimeout: time.Second,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if !strings.Contains(err.Error(), "TLS 证书加载失败") {
		t.Errorf("错误消息应包含 'TLS 证书加载失败'，实际 %q", err.Error())
	}
}

func TestConfig_Validate_NegativeTimeouts(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	base := &Config{
		TLSCertFile: certFile,
		TLSKeyFile:  keyFile,
	}

	tests := []struct {
		name   string
		modify func(*Config)
		want   string
	}{
		{
			name:   "ShutdownTimeout 负数",
			modify: func(c *Config) { c.ShutdownTimeout = -1 },
			want:   "关闭超时时间不能为负数",
		},
		{
			name:   "RequestTimeout 负数",
			modify: func(c *Config) { c.RequestTimeout = -1 },
			want:   "请求超时时间不能为负数",
		},
		{
			name:   "ReadTimeout 负数",
			modify: func(c *Config) { c.ReadTimeout = -1 },
			want:   "读取超时时间不能为负数",
		},
		{
			name:   "WriteTimeout 负数",
			modify: func(c *Config) { c.WriteTimeout = -1 },
			want:   "写入超时时间不能为负数",
		},
		{
			name:   "IdleTimeout 负数",
			modify: func(c *Config) { c.IdleTimeout = -1 },
			want:   "空闲超时时间不能为负数",
		},
		{
			name:   "MaxHeaderBytes 负数",
			modify: func(c *Config) { c.MaxHeaderBytes = -1 },
			want:   "最大请求头字节数不能为负数",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := *base
			tt.modify(&c)
			err := c.Validate()
			if err == nil {
				t.Fatal("期望返回错误，实际为 nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("错误消息应包含 %q，实际为 %q", tt.want, err.Error())
			}
		})
	}
}

func TestConfig_Validate_InvalidLogLevel(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	c := &Config{
		TLSCertFile:     certFile,
		TLSKeyFile:      keyFile,
		LogLevel:        "verbose",
		ShutdownTimeout: time.Second,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if !strings.Contains(err.Error(), "日志级别无效") {
		t.Errorf("错误消息应包含 '日志级别无效'，实际 %q", err.Error())
	}
}

func TestConfig_Validate_DefaultLogLevel(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	c := &Config{
		TLSCertFile:     certFile,
		TLSKeyFile:      keyFile,
		ShutdownTimeout: time.Second,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("期望校验通过，实际返回错误：%v", err)
	}
	if c.LogLevel != "info" {
		t.Errorf("期望默认 LogLevel 为 'info'，实际为 %q", c.LogLevel)
	}
}

func TestConfig_Validate_DefaultHealthPath(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	c := &Config{
		TLSCertFile:     certFile,
		TLSKeyFile:      keyFile,
		ShutdownTimeout: time.Second,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("期望校验通过，实际返回错误：%v", err)
	}
	if c.HealthPath != "/health" {
		t.Errorf("期望默认 HealthPath 为 '/health'，实际为 %q", c.HealthPath)
	}
}

func TestConfig_Validate_CustomHealthPath(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	c := &Config{
		TLSCertFile:     certFile,
		TLSKeyFile:      keyFile,
		HealthPath:      "/custom-health",
		ShutdownTimeout: time.Second,
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("期望校验通过，实际返回错误：%v", err)
	}
	if c.HealthPath != "/custom-health" {
		t.Errorf("期望 HealthPath 保持为 '/custom-health'，实际为 %q", c.HealthPath)
	}
}

func TestConfig_Validate_ValidLogLevels(t *testing.T) {
	certFile, keyFile := generateTestCert(t)

	levels := []string{"debug", "info", "warn", "error"}
	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			c := &Config{
				TLSCertFile:     certFile,
				TLSKeyFile:      keyFile,
				LogLevel:        level,
				ShutdownTimeout: time.Second,
			}
			if err := c.Validate(); err != nil {
				t.Errorf("期望日志级别 %q 校验通过，实际返回错误：%v", level, err)
			}
		})
	}
}
