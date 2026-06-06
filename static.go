package ginx

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// staticEntry 缓存单条静态文件服务配置，在 Start() 时注册到 gin.Engine。
type staticEntry struct {
	prefix string
	fs     http.FileSystem
}

// spaConfig 缓存 SPA 回退配置。
type spaConfig struct {
	fs        http.FileSystem
	indexPath string
}

// ServeStaticDir 从本地目录提供静态文件。
//
// 等价于 gin.Engine.Static(prefix, root)。
// root 必须是本地文件系统上已存在的目录。
func (s *Server) ServeStaticDir(prefix, root string) *Server {
	return s.ServeStaticFS(prefix, http.Dir(root))
}

// ServeStaticFS 从 http.FileSystem 提供静态文件。
//
// 配合 Go embed 包使用可将前端资源嵌入二进制：
//
//	//go:embed all:frontend/dist
//	var spaAssets embed.FS
//	distFS, _ := fs.Sub(spaAssets, "frontend/dist")
//	s.ServeStaticFS("/", http.FS(distFS))
//
// 等价于 gin.Engine.StaticFS(prefix, fs)。
func (s *Server) ServeStaticFS(prefix string, filesys http.FileSystem) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	s.staticEntries = append(s.staticEntries, staticEntry{
		prefix: prefix,
		fs:     filesys,
	})
	return s
}

// EnableSPA 启用 SPA 回退模式。
//
// 启用后，未匹配到任何路由的 GET/HEAD 请求将返回指定的 index 文件，
// 而非标准的 JSON 404 响应。非 GET/HEAD 请求仍返回标准 JSON 错误。
//
// indexPath 是相对于 fs 的文件路径，通常为 "index.html"。
// 重复调用会覆盖之前的配置。
//
// 使用示例：
//
//	s.ServeStaticFS("/", http.FS(distFS))
//	s.EnableSPA(http.FS(distFS), "index.html")
func (s *Server) EnableSPA(filesys http.FileSystem, indexPath string) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.Warn(context.Background(), "ginx：服务已启动，不允许修改配置")
		return s
	}
	s.spa = &spaConfig{
		fs:        filesys,
		indexPath: indexPath,
	}
	return s
}

// spaNoRoute 返回一个 SPA 模式的 NoRoute Handler。
// GET/HEAD 请求返回 index 文件，其他方法返回标准 JSON 404。
//
// 注意：不使用 c.FileFromFS()，因为 http.serveFile 会对以
// "/index.html" 结尾的 URL 触发 301 重定向（clean URL 机制）。
// 改为手动打开文件并通过 http.ServeContent 直接提供内容。
func spaNoRoute(filesys http.FileSystem, indexPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			file, err := filesys.Open(indexPath)
			if err != nil {
				writeJSON(c, http.StatusNotFound, "请求的资源不存在", nil)
				return
			}
			defer file.Close()

			stat, err := file.Stat()
			if err != nil {
				writeJSON(c, http.StatusInternalServerError, "服务器内部错误", nil)
				return
			}

			if stat.IsDir() {
				writeJSON(c, http.StatusNotFound, "请求的资源不存在", nil)
				return
			}

			http.ServeContent(c.Writer, c.Request, indexPath, stat.ModTime(), file)
			return
		}
		writeJSON(c, http.StatusNotFound, "请求的资源不存在", nil)
	}
}
