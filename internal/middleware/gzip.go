package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// compressReader оборачивает io.ReadCloser для распаковки тела запроса
type compressReader struct {
	r          io.ReadCloser
	gzipReader *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:          r,
		gzipReader: gzipReader,
	}, nil
}

func (c *compressReader) Read(p []byte) (n int, err error) {
	return c.gzipReader.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.gzipReader.Close(); err != nil {
		return err
	}
	return c.r.Close()
}

// shouldCompress проверяет, нужно ли сжимать ответ на основе Content-Type
func shouldCompress(contentType string) bool {
	// Извлекаем тип без параметров (например, "application/json; charset=utf-8" -> "application/json")
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return ct == "application/json" || ct == "text/html"
}

// gzipResponseWriter оборачивает http.ResponseWriter и решает сжимать или нет на основе Content-Type
type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter     *gzip.Writer
	wroteHeader    bool
	shouldCompress bool
	compressing    bool
}

func newGzipResponseWriter(w http.ResponseWriter) *gzipResponseWriter {
	return &gzipResponseWriter{
		ResponseWriter: w,
		gzipWriter:     gzip.NewWriter(w),
	}
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	// Проверяем Content-Type и решаем, нужно ли сжимать
	contentType := w.Header().Get("Content-Type")
	w.shouldCompress = shouldCompress(contentType)

	if w.shouldCompress && statusCode < 300 {
		w.Header().Set("Content-Encoding", "gzip")
		w.compressing = true
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.compressing {
		return w.gzipWriter.Write(data)
	}

	return w.ResponseWriter.Write(data)
}

func (w *gzipResponseWriter) Close() error {
	if w.compressing {
		return w.gzipWriter.Close()
	}
	return nil
}

// GzipMiddleware добавляет поддержку сжатия gzip для запросов и ответов
func GzipMiddleware(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Обработка входящих сжатых запросов
			if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
				cr, err := newCompressReader(r.Body)
				if err != nil {
					logger.Error("Failed to decompress request body",
						zap.Error(err),
						zap.String("uri", r.RequestURI),
						zap.String("method", r.Method),
						zap.String("remote_addr", r.RemoteAddr),
					)
					http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
					return
				}
				defer func() {
					if err := cr.Close(); err != nil {
						logger.Warn("Failed to close compress reader",
							zap.Error(err),
							zap.String("uri", r.RequestURI),
						)
					}
				}()
				r.Body = cr
			}

			// Проверяем, поддерживает ли клиент сжатие
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				// Клиент не поддерживает gzip, отправляем несжатый ответ
				next.ServeHTTP(w, r)
				return
			}

			// Создаем обертку для возможного сжатия ответа
			gzipWriter := newGzipResponseWriter(w)
			defer func() {
				if err := gzipWriter.Close(); err != nil {
					logger.Error("Failed to close gzip writer",
						zap.Error(err),
						zap.String("uri", r.RequestURI),
					)
				}
			}()

			next.ServeHTTP(gzipWriter, r)
		})
	}
}
