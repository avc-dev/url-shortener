package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

// compressString сжимает строку с помощью gzip
func compressString(s string) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	if _, err := gzipWriter.Write([]byte(s)); err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decompressBytes распаковывает данные gzip
func decompressBytes(data []byte) (string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer reader.Close()

	result, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func TestGzipMiddleware_CompressResponse(t *testing.T) {
	tests := []struct {
		name           string
		contentType    string
		acceptEncoding string
		body           string
		shouldCompress bool
	}{
		{
			name:           "compress JSON response",
			contentType:    "application/json",
			acceptEncoding: "gzip",
			body:           `{"result":"http://localhost:8080/abc123"}`,
			shouldCompress: true,
		},
		{
			name:           "compress JSON with charset",
			contentType:    "application/json; charset=utf-8",
			acceptEncoding: "gzip",
			body:           `{"result":"http://localhost:8080/abc123"}`,
			shouldCompress: true,
		},
		{
			name:           "compress HTML response",
			contentType:    "text/html",
			acceptEncoding: "gzip",
			body:           "<html><body>Hello World</body></html>",
			shouldCompress: true,
		},
		{
			name:           "compress HTML with charset",
			contentType:    "text/html; charset=utf-8",
			acceptEncoding: "gzip",
			body:           "<html><body>Hello World</body></html>",
			shouldCompress: true,
		},
		{
			name:           "do not compress without Accept-Encoding",
			contentType:    "application/json",
			acceptEncoding: "",
			body:           `{"result":"http://localhost:8080/abc123"}`,
			shouldCompress: false,
		},
		{
			name:           "do not compress text/plain",
			contentType:    "text/plain",
			acceptEncoding: "gzip",
			body:           "plain text response",
			shouldCompress: false,
		},
		{
			name:           "do not compress application/xml",
			contentType:    "application/xml",
			acceptEncoding: "gzip",
			body:           "<xml>data</xml>",
			shouldCompress: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовый logger
			logger := zaptest.NewLogger(t)

			// Создаем тестовый handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.body))
			})

			// Оборачиваем в gzip middleware
			wrappedHandler := GzipMiddleware(logger)(handler)

			// Создаем тестовый запрос
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			// Создаем recorder для захвата ответа
			rec := httptest.NewRecorder()

			// Выполняем запрос
			wrappedHandler.ServeHTTP(rec, req)

			// Проверяем результат
			if tt.shouldCompress {
				// Проверяем наличие заголовка Content-Encoding
				if rec.Header().Get("Content-Encoding") != "gzip" {
					t.Errorf("Expected Content-Encoding: gzip, got: %s", rec.Header().Get("Content-Encoding"))
				}

				// Проверяем, что тело сжато и может быть распаковано
				decompressed, err := decompressBytes(rec.Body.Bytes())
				if err != nil {
					t.Fatalf("Failed to decompress response: %v", err)
				}

				if decompressed != tt.body {
					t.Errorf("Decompressed body mismatch.\nExpected: %s\nGot: %s", tt.body, decompressed)
				}
			} else {
				// Проверяем отсутствие заголовка Content-Encoding
				if rec.Header().Get("Content-Encoding") == "gzip" {
					t.Error("Did not expect Content-Encoding: gzip")
				}

				// Проверяем, что тело не сжато
				if rec.Body.String() != tt.body {
					t.Errorf("Body mismatch.\nExpected: %s\nGot: %s", tt.body, rec.Body.String())
				}
			}
		})
	}
}

func TestGzipMiddleware_DecompressRequest(t *testing.T) {
	tests := []struct {
		name            string
		requestBody     string
		contentEncoding string
		compress        bool
		expectError     bool
	}{
		{
			name:            "decompress gzip request",
			requestBody:     `{"url":"https://practicum.yandex.ru"}`,
			contentEncoding: "gzip",
			compress:        true,
			expectError:     false,
		},
		{
			name:            "pass through non-gzip request",
			requestBody:     `{"url":"https://practicum.yandex.ru"}`,
			contentEncoding: "",
			compress:        false,
			expectError:     false,
		},
		{
			name:            "handle invalid gzip data",
			requestBody:     "not gzip data",
			contentEncoding: "gzip",
			compress:        false, // не сжимаем, чтобы отправить невалидные данные
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedBody string

			// Создаем тестовый logger
			logger := zaptest.NewLogger(t)

			// Создаем тестовый handler, который читает тело запроса
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("Failed to read request body: %v", err)
				}
				receivedBody = string(body)
				w.WriteHeader(http.StatusOK)
			})

			// Оборачиваем в gzip middleware
			wrappedHandler := GzipMiddleware(logger)(handler)

			// Подготавливаем тело запроса
			var requestBody io.Reader
			if tt.compress {
				compressed, err := compressString(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to compress test data: %v", err)
				}
				requestBody = bytes.NewReader(compressed)
			} else {
				requestBody = strings.NewReader(tt.requestBody)
			}

			// Создаем тестовый запрос
			req := httptest.NewRequest(http.MethodPost, "/", requestBody)
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}

			// Создаем recorder для захвата ответа
			rec := httptest.NewRecorder()

			// Выполняем запрос
			wrappedHandler.ServeHTTP(rec, req)

			// Проверяем результат
			if tt.expectError {
				if rec.Code != http.StatusBadRequest {
					t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rec.Code)
				}
			} else {
				if rec.Code != http.StatusOK {
					t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
				}

				if receivedBody != tt.requestBody {
					t.Errorf("Received body mismatch.\nExpected: %s\nGot: %s", tt.requestBody, receivedBody)
				}
			}
		})
	}
}

func TestGzipMiddleware_BothDirections(t *testing.T) {
	// Тест для проверки одновременной работы сжатия и распаковки
	expectedRequest := `{"url":"https://practicum.yandex.ru"}`
	expectedResponse := `{"result":"http://localhost:8080/abc123"}`

	var receivedBody string

	// Создаем тестовый logger
	logger := zaptest.NewLogger(t)

	// Создаем тестовый handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}
		receivedBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(expectedResponse))
	})

	// Оборачиваем в gzip middleware
	wrappedHandler := GzipMiddleware(logger)(handler)

	// Сжимаем тело запроса
	compressedRequest, err := compressString(expectedRequest)
	if err != nil {
		t.Fatalf("Failed to compress request: %v", err)
	}

	// Создаем тестовый запрос
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(compressedRequest))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")

	// Создаем recorder для захвата ответа
	rec := httptest.NewRecorder()

	// Выполняем запрос
	wrappedHandler.ServeHTTP(rec, req)

	// Проверяем статус
	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rec.Code)
	}

	// Проверяем, что запрос был распакован
	if receivedBody != expectedRequest {
		t.Errorf("Request body mismatch.\nExpected: %s\nGot: %s", expectedRequest, receivedBody)
	}

	// Проверяем, что ответ сжат
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Expected Content-Encoding: gzip in response")
	}

	// Распаковываем ответ
	decompressedResponse, err := decompressBytes(rec.Body.Bytes())
	if err != nil {
		t.Fatalf("Failed to decompress response: %v", err)
	}

	if decompressedResponse != expectedResponse {
		t.Errorf("Response body mismatch.\nExpected: %s\nGot: %s", expectedResponse, decompressedResponse)
	}
}

func TestShouldCompress(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"TEXT/HTML", true},        // case insensitive
		{"APPLICATION/JSON", true}, // case insensitive
		{"text/plain", false},
		{"application/xml", false},
		{"image/png", false},
		{"application/octet-stream", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := shouldCompress(tt.contentType)
			if result != tt.expected {
				t.Errorf("shouldCompress(%q) = %v, expected %v", tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestGzipMiddleware_LogsErrors(t *testing.T) {
	t.Run("logs decompression error", func(t *testing.T) {
		// Создаем тестовый logger с наблюдателем
		observedZapCore, observedLogs := observer.New(zapcore.ErrorLevel)
		logger := zap.New(observedZapCore)

		// Создаем простой handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Оборачиваем в gzip middleware
		wrappedHandler := GzipMiddleware(logger)(handler)

		// Отправляем невалидные gzip данные
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("invalid gzip data"))
		req.Header.Set("Content-Encoding", "gzip")

		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)

		// Проверяем, что ошибка залогирована
		if observedLogs.Len() == 0 {
			t.Error("Expected error to be logged, but no logs found")
		}

		// Проверяем, что ошибка содержит правильное сообщение
		logEntries := observedLogs.All()
		found := false
		for _, entry := range logEntries {
			if entry.Message == "Failed to decompress request body" {
				found = true
				// Проверяем, что залогированы детали
				hasError := false
				for _, field := range entry.Context {
					if field.Key == "error" {
						hasError = true
						break
					}
				}
				if !hasError {
					t.Error("Expected error field in log entry")
				}
				break
			}
		}
		if !found {
			t.Error("Expected log entry with message 'Failed to decompress request body'")
		}

		// Проверяем статус код
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}
