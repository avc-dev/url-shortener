package service

import (
	"testing"

	"github.com/avc-dev/url-shortener/internal/config"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/repository"
	"github.com/avc-dev/url-shortener/internal/store"
)

// BenchmarkCodeGeneratorGenerateCode измеряет скорость генерации одного кода.
func BenchmarkCodeGeneratorGenerateCode(b *testing.B) {
	gen := NewCodeGenerator()
	b.ReportAllocs()
	for b.Loop() {
		_ = gen.GenerateCode()
	}
}

// BenchmarkCodeGeneratorGenerateBatchCodes измеряет генерацию пакета из 100 кодов.
func BenchmarkCodeGeneratorGenerateBatchCodes(b *testing.B) {
	gen := NewCodeGenerator()
	b.ReportAllocs()
	for b.Loop() {
		_ = gen.GenerateBatchCodes(100)
	}
}

// BenchmarkURLServiceCreateShortURL измеряет полный путь создания короткого URL
// через сервисный слой с реальным in-memory хранилищем.
// Каждая итерация создаёт уникальный URL (счётчик в коде), поэтому тест
// измеряет сценарий «новая запись», а не «дубликат».
func BenchmarkURLServiceCreateShortURL(b *testing.B) {
	st := store.NewStore()
	repo := repository.New(st)
	cfg := config.NewDefaultConfig()
	svc := NewURLService(repo, cfg)

	b.ReportAllocs()
	n := 0
	for b.Loop() {
		n++
		url := model.URL("https://example.com/bench/" + model.URL(string(rune('a'+n%26))))
		_, _, _ = svc.CreateShortURL(url, "user1")
	}
}

// BenchmarkURLServiceCreateShortURL_Duplicate измеряет сценарий, когда URL уже
// существует и сервис возвращает существующий код без создания новой записи.
func BenchmarkURLServiceCreateShortURL_Duplicate(b *testing.B) {
	st := store.NewStore()
	repo := repository.New(st)
	cfg := config.NewDefaultConfig()
	svc := NewURLService(repo, cfg)

	const existingURL = model.URL("https://example.com/existing")
	_, _, _ = svc.CreateShortURL(existingURL, "user1")

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _, _ = svc.CreateShortURL(existingURL, "user1")
	}
}

// BenchmarkURLServiceCreateShortURLsBatch измеряет пакетное создание 10 коротких URL.
func BenchmarkURLServiceCreateShortURLsBatch(b *testing.B) {
	st := store.NewStore()
	repo := repository.New(st)
	cfg := config.NewDefaultConfig()
	svc := NewURLService(repo, cfg)

	urls := make([]model.URL, 10)
	for i := range urls {
		urls[i] = model.URL("https://example.com/batch-bench/" + model.URL(string(rune('a'+i))))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		// Создаём новый набор URL каждый раз, чтобы избежать конфликтов
		batch := make([]model.URL, len(urls))
		copy(batch, urls)
		_, _ = svc.CreateShortURLsBatch(batch, "user2")
	}
}
