package store

import (
	"fmt"
	"testing"

	"github.com/avc-dev/url-shortener/internal/model"
)

// BenchmarkStoreWrite измеряет запись одной новой записи в in-memory хранилище.
func BenchmarkStoreWrite(b *testing.B) {
	s := NewStore()
	b.ReportAllocs()
	n := 0
	for b.Loop() {
		n++
		key := model.Code(fmt.Sprintf("code%08d", n))
		val := model.URL(fmt.Sprintf("https://example.com/%d", n))
		_ = s.Write(key, val, "user1")
	}
}

// BenchmarkStoreRead измеряет чтение существующего ключа (горячий путь).
func BenchmarkStoreRead(b *testing.B) {
	s := NewStore()
	_ = s.Write("benchkey", "https://example.com/target", "user1")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = s.Read("benchkey")
	}
}

// BenchmarkStoreCreateOrGetURL_New измеряет создание нового URL (запись в хранилище).
func BenchmarkStoreCreateOrGetURL_New(b *testing.B) {
	s := NewStore()
	b.ReportAllocs()
	n := 0
	for b.Loop() {
		n++
		code := model.Code(fmt.Sprintf("code%08d", n))
		url := model.URL(fmt.Sprintf("https://example.com/%d", n))
		_, _, _ = s.CreateOrGetURL(code, url, "user1")
	}
}

// BenchmarkStoreCreateOrGetURL_Existing измеряет сценарий, когда URL уже существует
// и хранилище возвращает существующий код (O(n) поиск по карте в базовой версии).
func BenchmarkStoreCreateOrGetURL_Existing(b *testing.B) {
	s := NewStore()

	// Заполняем 1000 записей, чтобы O(n) поиск был заметен в профиле
	for i := range 1000 {
		code := model.Code(fmt.Sprintf("pre%06d", i))
		url := model.URL(fmt.Sprintf("https://example.com/pre/%d", i))
		_ = s.Write(code, url, "user1")
	}
	targetURL := model.URL("https://example.com/pre/999")

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _, _ = s.CreateOrGetURL("newcode", targetURL, "user1")
	}
}

// BenchmarkStoreGetURLsByUserID измеряет получение всех URL пользователя —
// именно здесь url.JoinPath вызывается на каждый элемент результата.
func BenchmarkStoreGetURLsByUserID(b *testing.B) {
	s := NewStore()
	const numURLs = 100

	for i := range numURLs {
		code := model.Code(fmt.Sprintf("usr%06d", i))
		url := model.URL(fmt.Sprintf("https://example.com/user-path/%d", i))
		_ = s.Write(code, url, "benchuser")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = s.GetURLsByUserID("benchuser", "http://localhost:8080")
	}
}

// BenchmarkStoreIsCodeUnique измеряет проверку уникальности кода.
func BenchmarkStoreIsCodeUnique(b *testing.B) {
	s := NewStore()
	_ = s.Write("exists", "https://example.com", "u1")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = s.IsCodeUnique("exists")
		_ = s.IsCodeUnique("notexists")
	}
}
