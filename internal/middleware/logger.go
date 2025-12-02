package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// responseWriter is a wrapper around http.ResponseWriter that captures
// the status code and response size for logging purposes.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// Logger returns a middleware that logs HTTP requests using zap logger.
// It logs the method, URI, status code, duration, and response size for each request.
func Logger(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := newResponseWriter(w)

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("uri", r.RequestURI),
				zap.Int("status", wrapped.statusCode),
				zap.Duration("duration", duration),
				zap.Int64("size", wrapped.written),
				zap.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}
