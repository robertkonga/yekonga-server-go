package yekonga

import (
	"embed"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/console"
)

//go:embed static/*
var StaticFS embed.FS

// Response represents an HTTP response wrapper
type Response struct {
	httpResponseWriter *http.ResponseWriter
	staticConfig       []*StaticConfig
	request            *Request
}

// Response methods
func (res *Response) Status(code int) {
	(*res.httpResponseWriter).WriteHeader(code)
}

func (res *Response) Header(key string, value string) {
	(*res.httpResponseWriter).Header().Set(key, value)
}

func (res *Response) Abort(code int, message string) {
	var contentUrl string
	res.Status(code)

	isJson := (strings.Contains(res.request.GetHeader("content-type"), "json"))
	switch code {
	case 400:
		if helper.IsEmpty(message) {
			message = "400 Bad Request"
		}

		contentUrl = "static/400.html"
	case 401:
		if helper.IsEmpty(message) {
			message = "401 Unauthorized"
		}

		contentUrl = "static/401.html"
	case 403:
		if helper.IsEmpty(message) {
			message = "403 forbidden"
		}

		contentUrl = "static/403.html"
	case 404:
		if helper.IsEmpty(message) {
			message = "404 Page Not Found"
		}

		contentUrl = "static/404.html"
	case 500:
		if helper.IsEmpty(message) {
			message = "500 Internal Server Error"
		}

		contentUrl = "static/500.html"
	}

	if isJson {
		res.Json(map[string]interface{}{
			"error": message,
		})
		return
	} else if helper.IsNotEmpty(contentUrl) {
		content, err := StaticFS.ReadFile(contentUrl)
		if err != nil {
			res.Text(message)
		} else {
			res.Html(string(content))
		}
		return
	}

	res.Text(message)
}

func (res *Response) Text(data string) {
	(*res.httpResponseWriter).Header().Set("Content-Type", "text/plain")
	(*res.httpResponseWriter).Write([]byte(data))
}

func (res *Response) Byte(data []byte) {
	(*res.httpResponseWriter).Write(data)
}

func (res *Response) Html(data string) {
	(*res.httpResponseWriter).Header().Set("Content-Type", "text/html")
	(*res.httpResponseWriter).Write([]byte(data))
}

func (res *Response) Json(data interface{}) {
	(*res.httpResponseWriter).Header().Set("Content-Type", "application/json")
	json.NewEncoder(*res.httpResponseWriter).Encode(data)
}

func (res *Response) File(file string) {
	count := len(res.staticConfig)

	if helper.FileExists(file) {
		// Set cache headers
		(*res.httpResponseWriter).Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", 1))
		// Serve the file
		http.ServeFile(*res.httpResponseWriter, res.request.HttpRequest, file)
		return
	} else {
		for i := 0; i < count; i++ {
			static := res.staticConfig[i]

			if static != nil {
				filePath := filepath.Join(static.Directory, file)

				if helper.FileExists(filePath) {
					// Set cache headers
					(*res.httpResponseWriter).Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", static.CacheMaxAge))
					// Serve the file
					http.ServeFile(*res.httpResponseWriter, res.request.HttpRequest, filePath)
					return
				}
			}
		}
	}

	res.Abort(404, "")
}

func (res *Response) Download(filename string, name string) {
	CacheMaxAge := false

	console.Warn("filename", filename)

	if helper.FileExists(filename) {
		if helper.IsEmpty(name) {
			name = filepath.Base(filename)
		}
		// === 2. Validate file exists ===
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			http.Error(*res.httpResponseWriter, "Config file not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(*res.httpResponseWriter, "Error accessing file", http.StatusInternalServerError)
			return
		}

		// === 3. Open file ===
		file, err := os.Open(filename)
		if err != nil {
			http.Error(*res.httpResponseWriter, "Failed to open file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// === 4. Get file info ===
		stat, err := file.Stat()
		if err != nil {
			http.Error(*res.httpResponseWriter, "Failed to read file info", http.StatusInternalServerError)
			return
		}

		// === 5. Set headers ===
		// Force download (not browser preview)
		(*res.httpResponseWriter).Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))

		// MIME type (auto-detect)
		if mimeType := mime.TypeByExtension(filepath.Ext(name)); mimeType != "" {
			(*res.httpResponseWriter).Header().Set("Content-Type", mimeType)
		} else {
			(*res.httpResponseWriter).Header().Set("Content-Type", "application/octet-stream")
		}

		if !CacheMaxAge {
			// Cache control: short cache or no-cache
			(*res.httpResponseWriter).Header().Set("Cache-Control", "max-age=1, must-revalidate") // or "no-cache, no-store"
		}

		// Optional security headers
		(*res.httpResponseWriter).Header().Set("X-Content-Type-Options", "nosniff")

		// === 6. Serve file efficiently (zero-copy) ===
		http.ServeContent(*res.httpResponseWriter, res.request.HttpRequest, filename, stat.ModTime(), file)
		return
	}

	res.Abort(404, "")
}
