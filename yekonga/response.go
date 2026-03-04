package yekonga

import (
	"compress/gzip"
	"embed"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/robertkonga/yekonga-server-go/helper"
)

//go:embed static/*
var StaticFS embed.FS

type MIMEHeader map[string][]string

// Response represents an HTTP response wrapper
type Response struct {
	httpResponseWriter *http.ResponseWriter
	staticConfig       []*StaticConfig
	request            *Request
	headers            MIMEHeader
	statusCode         int
	gz                 *gzip.Writer
}

// Response methods
func (res *Response) Status(code int) {
	res.statusCode = code
}

func (res *Response) Header() http.Header {
	return (*res.httpResponseWriter).Header()
}

func (res *Response) ResetHeaders() {
	for k, v := range res.headers {
		for _, s := range v {
			(*res.httpResponseWriter).Header().Set(k, s)
		}
	}
}

func (res *Response) SetHeader(key string, value string) {
	if v, ok := res.headers[key]; !ok {
		res.headers[key] = append(v, value)
	} else {
		res.headers[key] = []string{value}
	}
}

func (res *Response) WriteHeader(statusCode int) {
	(*res.httpResponseWriter).WriteHeader(statusCode)
}

func (res *Response) Abort(code int, message string) {
	var contentUrl string

	isJson := (strings.Contains(res.request.GetHeader("content-type"), "json"))

	if !isJson {
		isJson = (strings.Contains(res.request.GetHeader("accept"), "json"))
	}

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

	// console.Error("Abort:", code, message)
	res.Status(code)

	if isJson {
		res.Json(map[string]interface{}{
			"status": code,
			"error":  message,
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

func (res *Response) acceptsGzip() bool {
	for _, enc := range strings.Split(res.request.HttpRequest.Header.Get("accept-encoding"), ",") {
		if strings.TrimSpace(strings.SplitN(enc, ";", 2)[0]) == "gzip" {
			return true
		}
	}
	return false
}

func (res *Response) initGzip(w *http.ResponseWriter) error {
	if res.gz != nil {
		return nil
	}

	(*w).Header().Set("content-encoding", "gzip")
	(*w).Header().Set("vary", "accept-encoding")

	gz := gzip.NewWriter(*w)
	res.gz = gz

	return nil
}

func (res *Response) Write(data []byte) (int, error) {
	res.SetHeader("yekonga-application", "Yesu")
	res.ResetHeaders()

	if res.acceptsGzip() {
		if err := res.initGzip(res.httpResponseWriter); err == nil {
			(*res.httpResponseWriter).WriteHeader(res.statusCode)

			return res.gz.Write(data)
		}
		// initGzip failed — fall through to uncompressed (headers not yet set)
	}
	return (*res.httpResponseWriter).Write(data)
}

// Close must be called when the response is done to flush the gzip stream.
func (res *Response) Close() error {
	if res.gz != nil {
		return res.gz.Close()
	}
	return nil
}

func (res *Response) Text(data string) {
	res.SetHeader("content-type", "text/plain")

	res.Write([]byte(data))
}

func (res *Response) Byte(data []byte) {
	res.Write([]byte(data))
}

func (res *Response) Send(data string) {
	res.Write([]byte(data))
}

func (res *Response) Html(data string) {
	res.SetHeader("Content-Type", "text/html")

	res.Write([]byte(data))
}

func (res *Response) Json(data interface{}) {
	res.SetHeader("Content-Type", "application/json")
	res.ResetHeaders()

	// json.NewEncoder((*res.httpResponseWriter)).Encode(data)
	res.Write([]byte(helper.ToJson(data)))
}

func (res *Response) File(file string) {
	count := len(res.staticConfig)

	if helper.FileExists(file) {
		// Set cache headers
		res.SetHeader("cache-control", fmt.Sprintf("max-age=%d", 1))
		res.ResetHeaders()

		// Serve the file
		http.ServeFile(res, res.request.HttpRequest, file)
		return
	} else {
		for i := 0; i < count; i++ {
			static := res.staticConfig[i]

			if static != nil {
				filePath := filepath.Join(static.Directory, file)

				if helper.FileExists(filePath) {
					// Set cache headers
					res.SetHeader("cache-control", fmt.Sprintf("max-age=%d", static.CacheMaxAge))
					res.ResetHeaders()
					// Serve the file
					http.ServeFile((*res.httpResponseWriter), res.request.HttpRequest, filePath)
					return
				}
			}
		}
	}

	res.Abort(404, "")
}

func (res *Response) Download(filename string, name string) {
	CacheMaxAge := false

	if helper.FileExists(filename) {
		if helper.IsEmpty(name) {
			name = filepath.Base(filename)
		}
		// === 2. Validate file exists ===
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			http.Error(res, "Config file not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(res, "Error accessing file", http.StatusInternalServerError)
			return
		}

		// === 3. Open file ===
		file, err := os.Open(filename)
		if err != nil {
			http.Error(res, "Failed to open file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// === 4. Get file info ===
		stat, err := file.Stat()
		if err != nil {
			http.Error(res, "Failed to read file info", http.StatusInternalServerError)
			return
		}

		// === 5. Set headers ===
		// Force download (not browser preview)
		res.SetHeader("content-disposition", fmt.Sprintf(`attachment; filename="%s"`, name))

		// MIME type (auto-detect)
		if mimeType := mime.TypeByExtension(filepath.Ext(name)); mimeType != "" {
			res.SetHeader("content-type", mimeType)
		} else {
			res.SetHeader("content-type", "application/octet-stream")
		}

		if !CacheMaxAge {
			// Cache control: short cache or no-cache
			res.SetHeader("cache-control", "max-age=1, must-revalidate") // or "no-cache, no-store"
		}

		// Optional security headers
		res.SetHeader("x-content-type-options", "nosniff")
		res.ResetHeaders()

		// === 6. Serve file efficiently (zero-copy) ===
		http.ServeContent(res, res.request.HttpRequest, filename, stat.ModTime(), file)
		return
	}

	res.Abort(404, "")
}

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer *gzip.Writer
}

func (g gzipResponseWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}
