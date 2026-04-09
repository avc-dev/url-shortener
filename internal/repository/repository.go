// Package repository реализует адаптер над хранилищем.
// Repository оборачивает Store, добавляя форматирование ошибок,
// и предоставляет единый интерфейс для usecase-слоя.
package repository

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/model"
)

// Store — интерфейс низкоуровневого хранилища, который должны реализовывать
// все конкретные бэкенды (in-memory, file, postgres).
type Store interface {
	// Read возвращает оригинальный URL по короткому коду.
	Read(key model.Code) (model.URL, error)
	// Write сохраняет пару код→URL с привязкой к пользователю.
	Write(key model.Code, value model.URL, userID string) error
	// WriteBatch сохраняет несколько пар код→URL для одного пользователя.
	WriteBatch(urls map[model.Code]model.URL, userID string) error
	// CreateOrGetURL атомарно создаёт запись или возвращает код уже существующего URL.
	// Второй возвращаемый параметр true означает, что запись была создана.
	CreateOrGetURL(code model.Code, url model.URL, userID string) (model.Code, bool, error)
	// IsCodeUnique возвращает true, если код ещё не занят.
	IsCodeUnique(code model.Code) bool
	// GetURLsByUserID возвращает все короткие ссылки пользователя с полными URL.
	GetURLsByUserID(userID string, baseURL string) ([]model.UserURLResponse, error)
	// DeleteURLsBatch помечает несколько кодов как удалённые для данного пользователя.
	DeleteURLsBatch(codes []model.Code, userID string) error
	// IsURLOwnedByUser проверяет, что код принадлежит указанному пользователю.
	IsURLOwnedByUser(code model.Code, userID string) bool
	// GetStats возвращает количество активных URL и уникальных пользователей.
	GetStats() (urlCount int, userCount int, err error)
}

// Repository адаптирует Store к интерфейсу, ожидаемому usecase-слоем.
// Все методы оборачивают ошибки с контекстом для удобства отладки.
type Repository struct {
	underlying Store
}

// New создаёт новый Repository поверх переданного Store.
func New(underlying Store) *Repository {
	return &Repository{underlying}
}

// IsCodeUnique проверяет, свободен ли код в хранилище.
func (r Repository) IsCodeUnique(code model.Code) bool {
	return r.underlying.IsCodeUnique(code)
}

// Write сохраняет пару код→URL с привязкой к пользователю.
func (r Repository) Write(code model.Code, url model.URL, userID string) error {
	err := r.underlying.Write(code, url, userID)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}

// CreateOrGetURL атомарно создаёт запись или возвращает код существующего URL.
func (r Repository) CreateOrGetURL(code model.Code, url model.URL, userID string) (model.Code, bool, error) {
	finalCode, created, err := r.underlying.CreateOrGetURL(code, url, userID)
	if err != nil {
		return "", false, fmt.Errorf("failed to create or get URL: %w", err)
	}

	return finalCode, created, nil
}

// CreateURLsBatch сохраняет несколько пар код→URL для одного пользователя.
func (r Repository) CreateURLsBatch(urls map[model.Code]model.URL, userID string) error {
	err := r.underlying.WriteBatch(urls, userID)
	if err != nil {
		return fmt.Errorf("failed to create URLs batch: %w", err)
	}
	return nil
}

// GetURLsByUserID возвращает все короткие ссылки пользователя с полными URL.
func (r Repository) GetURLsByUserID(userID string, baseURL string) ([]model.UserURLResponse, error) {
	urls, err := r.underlying.GetURLsByUserID(userID, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get URLs by user ID: %w", err)
	}
	return urls, nil
}

// DeleteURLsBatch помечает несколько URL как удалённые для данного пользователя.
func (r Repository) DeleteURLsBatch(codes []model.Code, userID string) error {
	err := r.underlying.DeleteURLsBatch(codes, userID)
	if err != nil {
		return fmt.Errorf("failed to delete URLs batch: %w", err)
	}
	return nil
}

// IsURLOwnedByUser проверяет, что код принадлежит указанному пользователю.
func (r Repository) IsURLOwnedByUser(code model.Code, userID string) bool {
	return r.underlying.IsURLOwnedByUser(code, userID)
}

// GetStats возвращает количество активных URL и уникальных пользователей.
func (r Repository) GetStats() (int, int, error) {
	urlCount, userCount, err := r.underlying.GetStats()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get stats: %w", err)
	}
	return urlCount, userCount, nil
}
