package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// gzipWriter wraps gin.ResponseWriter with a gzip.Writer
type gzipWriter struct {
	gin.ResponseWriter
	gw *gzip.Writer
}

func (w *gzipWriter) Write(data []byte) (int, error) {
	// Ensure content-type is set to avoid browsers misinterpreting content
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", http.DetectContentType(data))
	}
	return w.gw.Write(data)
}

func (w *gzipWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// Gzip is a simple gzip compression middleware for JSON/text API responses.
// It compresses responses when the client sends "Accept-Encoding: gzip".
func Gzip() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only gzip if client accepts it
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// Skip compression for WebSocket and SSE
		if strings.Contains(strings.ToLower(c.GetHeader("Connection")), "upgrade") ||
			strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "text/event-stream") {
			c.Next()
			return
		}

		// Set headers for gzip response
		h := c.Writer.Header()
		h.Set("Content-Encoding", "gzip")
		h.Add("Vary", "Accept-Encoding")
		h.Del("Content-Length") // length is unknown due to streaming compression

		// Replace the writer
		gw := gzip.NewWriter(c.Writer)
		defer func() {
			// Ensure the writer is closed to flush remaining data
			_ = gw.Close()
		}()

		gzw := &gzipWriter{ResponseWriter: c.Writer, gw: gw}
		c.Writer = gzw

		c.Next()

		// If the handler wrote nothing, close gracefully
		if flusher, ok := c.Writer.(interface{ Flush() }); ok {
			flusher.Flush()
		}
	}
}
