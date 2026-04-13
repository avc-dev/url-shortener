// Package store реализует хранилища для коротких URL:
// in-memory, файловое и PostgreSQL.
package store

import (
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"

	"github.com/avc-dev/url-shortener/internal/model"
)

var (
	ErrNotFound          = errors.New("key not found")
	ErrAlreadyExists     = errors.New("key already exists")
	ErrCodeAlreadyExists = errors.New("code already exists")
	ErrURLAlreadyExists  = errors.New("URL already exists")
	ErrURLDeleted        = errors.New("URL deleted")
)

// URLMap представляет маппинг коротких кодов на оригинальные URL
type URLMap = map[model.Code]model.URL

type Store struct {
	store      URLMap
	userMap    map[model.Code]string    // code -> userID mapping
	deletedMap map[model.Code]bool      // code -> is_deleted mapping
	urlIndex   map[model.URL]model.Code // reverse index: url -> code (O(1) lookup)
	mutex      sync.Mutex
}

func NewStore() *Store {
	return &Store{
		store:      make(URLMap),
		userMap:    make(map[model.Code]string),
		deletedMap: make(map[model.Code]bool),
		urlIndex:   make(map[model.URL]model.Code),
		mutex:      sync.Mutex{},
	}
}

func (s *Store) Read(key model.Code) (model.URL, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	value, ok := s.store[key]

	if !ok {
		return "", fmt.Errorf("key %s: %w", key, ErrNotFound)
	}

	// Проверяем, не удалён ли URL
	if s.deletedMap[key] {
		return "", fmt.Errorf("key %s: %w", key, ErrURLDeleted)
	}

	return value, nil
}

func (s *Store) Write(key model.Code, value model.URL, userID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Проверяем существование ключа напрямую, без вызова Read (чтобы избежать deadlock)
	if _, exists := s.store[key]; exists {
		return fmt.Errorf("code %s: %w", key, ErrCodeAlreadyExists)
	}

	s.store[key] = value
	s.userMap[key] = userID
	s.deletedMap[key] = false
	s.urlIndex[value] = key

	return nil
}

// InitializeWith инициализирует хранилище данными (без проверки на существование)
// Используется для массовой загрузки данных, например, из файла
func (s *Store) InitializeWith(data URLMap, userData map[model.Code]string, deletedData map[model.Code]bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	maps.Copy(s.store, data)
	maps.Copy(s.userMap, userData)
	if deletedData != nil {
		maps.Copy(s.deletedMap, deletedData)
	}
	// Перестраиваем обратный индекс из загруженных данных
	for code, url := range data {
		s.urlIndex[url] = code
	}
}

// WriteBatch сохраняет несколько пар код-URL в хранилище атомарно
func (s *Store) WriteBatch(urls URLMap, userID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Проверяем существование всех кодов перед вставкой
	for code := range urls {
		if _, exists := s.store[code]; exists {
			return fmt.Errorf("code %s: %w", code, ErrCodeAlreadyExists)
		}
	}

	// Вставляем все записи
	for code, url := range urls {
		s.store[code] = url
		s.userMap[code] = userID
		s.deletedMap[code] = false
		s.urlIndex[url] = code
	}

	return nil
}

// CreateOrGetURL создает новую запись или возвращает код существующей для данного URL.
// Использует обратный индекс urlIndex для O(1) поиска дубликата вместо O(n) перебора.
func (s *Store) CreateOrGetURL(code model.Code, url model.URL, userID string) (model.Code, bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// O(1) проверка через обратный индекс
	if existingCode, found := s.urlIndex[url]; found {
		// Обновляем userID для существующего кода
		s.userMap[existingCode] = userID
		return existingCode, false, nil // false = не создана новая запись
	}

	// Проверяем, свободен ли код
	if _, exists := s.store[code]; exists {
		return "", false, fmt.Errorf("code %s: %w", code, ErrCodeAlreadyExists)
	}

	// Создаем новую запись
	s.store[code] = url
	s.userMap[code] = userID
	s.deletedMap[code] = false
	s.urlIndex[url] = code
	return code, true, nil // true = создана новая запись
}

// IsCodeUnique проверяет, свободен ли код в хранилище
func (s *Store) IsCodeUnique(code model.Code) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, exists := s.store[code]
	return !exists
}

// GetCodeByURL возвращает код для существующего URL.
// O(1) поиск через обратный индекс urlIndex.
func (s *Store) GetCodeByURL(url model.URL) (model.Code, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if code, found := s.urlIndex[url]; found {
		return code, nil
	}

	return "", fmt.Errorf("URL not found: %w", ErrNotFound)
}

// GetURLsByUserID возвращает URL пользователя (исключая удалённые).
// Вместо url.JoinPath (≥3 аллокации/вызов) использует простую конкатенацию строк (1 аллокация).
// Результирующий срез предварительно выделяется под максимально возможный размер.
func (s *Store) GetURLsByUserID(userID string, baseURL string) ([]model.UserURLResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Нормализуем baseURL один раз перед циклом
	base := strings.TrimRight(baseURL, "/") + "/"

	// Считаем количество активных URL пользователя для предварительного выделения среза
	count := 0
	for code, storedUserID := range s.userMap {
		if storedUserID == userID && !s.deletedMap[code] {
			count++
		}
	}

	urls := make([]model.UserURLResponse, 0, count)

	for code, storedUserID := range s.userMap {
		if storedUserID == userID {
			// Проверяем, не удалён ли URL
			if s.deletedMap[code] {
				continue // Пропускаем удалённые URL
			}

			// Получаем оригинальный URL
			originalURL, exists := s.store[code]
			if !exists {
				continue // Несогласованность данных, пропускаем
			}

			// Формируем полный короткий URL: одна аллокация вместо ≥3 у url.JoinPath
			urls = append(urls, model.UserURLResponse{
				ShortURL:    base + string(code),
				OriginalURL: string(originalURL),
			})
		}
	}

	return urls, nil
}

// IsURLOwnedByUser проверяет, принадлежит ли URL указанному пользователю
func (s *Store) IsURLOwnedByUser(code model.Code, userID string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	storedUserID, exists := s.userMap[code]
	return exists && storedUserID == userID && !s.deletedMap[code]
}

// GetStats возвращает количество сокращённых URL (не удалённых) и уникальных пользователей
func (s *Store) GetStats() (model.Stats, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var stats model.Stats
	for code := range s.store {
		if !s.deletedMap[code] {
			stats.URLCount++
		}
	}

	seen := make(map[string]struct{}, len(s.userMap))
	for _, userID := range s.userMap {
		if userID != "" {
			seen[userID] = struct{}{}
		}
	}
	stats.UserCount = len(seen)

	return stats, nil
}

// DeleteURLsBatch помечает несколько URL как удалённые для указанного пользователя
func (s *Store) DeleteURLsBatch(codes []model.Code, userID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, code := range codes {
		// Проверяем, что URL принадлежит пользователю
		storedUserID, exists := s.userMap[code]
		if !exists || storedUserID != userID {
			continue // Пропускаем, если URL не существует или не принадлежит пользователю
		}

		// Помечаем как удалённый
		s.deletedMap[code] = true
	}

	return nil
}
