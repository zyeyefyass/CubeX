package webui

import (
	"compress/gzip"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

// resolveBuiltinMimeType 为常用静态后缀提供内置兜底，防止极简容器缺失 /etc/mime.types 时响应头为空
func resolveBuiltinMimeType(ext string) string {
	switch strings.ToLower(ext) {
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		return "application/json; charset=utf-8"
	case ".ico":
		return "image/x-icon"
	case ".woff2":
		return "font/woff2"
	case ".woff":
		return "font/woff"
	case ".ttf":
		return "font/ttf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return mime.TypeByExtension(ext)
	}
}

// PrecompressedFileServer 提供基于 embed.FS 的预压缩静态资产零拷贝直出服务
type PrecompressedFileServer struct {
	fsys fs.FS
}

// NewPrecompressedFileServer 创建预压缩静态资产服务引擎
func NewPrecompressedFileServer(fsys fs.FS) *PrecompressedFileServer {
	return &PrecompressedFileServer{fsys: fsys}
}

func (s *PrecompressedFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	upath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if upath == "" || upath == "." {
		upath = "index.html"
	}

	// 1. 设定准确的 Content-Type 与缓存策略（带 Hash 的 assets 强缓存 1 年，index.html 禁缓存）
	ext := filepath.Ext(upath)
	if ctype := resolveBuiltinMimeType(ext); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	if strings.HasPrefix(upath, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}

	acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

	// 2. 优先查找预压缩的 .gz 文件（0 CPU 压缩开销直接透传）
	if gzFile, err := s.fsys.Open(upath + ".gz"); err == nil {
		defer gzFile.Close()
		w.Header().Set("Vary", "Accept-Encoding")

		if acceptsGzip {
			w.Header().Set("Content-Encoding", "gzip")
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
				return
			}
			_, _ = io.Copy(w, gzFile)
			return
		}

		// 兼容极少数不支持 gzip 的客户端：流式解压输出
		gzReader, err := gzip.NewReader(gzFile)
		if err == nil {
			defer gzReader.Close()
			_, _ = io.Copy(w, gzReader)
			return
		}
	}

	// 3. 回退查找未压缩原文件（如小图标、字体等）
	if rawFile, err := s.fsys.Open(upath); err == nil {
		defer rawFile.Close()
		if stat, err := rawFile.Stat(); err == nil && !stat.IsDir() {
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
				return
			}
			_, _ = io.Copy(w, rawFile)
			return
		}
	}

	// 4. SPA 前端路由兜底回退至 index.html.gz 或 index.html
	if upath != "index.html" {
		r.URL.Path = "/index.html"
		s.ServeHTTP(w, r)
		return
	}

	http.NotFound(w, r)
}
