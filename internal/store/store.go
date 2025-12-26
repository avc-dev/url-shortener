package store

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"sync"

	"github.com/avc-dev/url-shortener/internal/model"
)

var (
	ErrNotFound          = errors.New("key not found")
	ErrAlreadyExists     = errors.New("key already exists")
	ErrCodeAlreadyExists = errors.New("code already exists")
	ErrURLAlreadyExists  = errors.New("URL already exists")
)

// URLMap представляет маппинг коротких кодов на оригинальные URL
type URLMap = map[model.Code]model.URL

type Store struct {
	store   URLMap
	userMap map[model.Code]string // code -> userID mapping
	mutex   sync.Mutex
}

func NewStore() *Store {
	return &Store{
		store:   make(URLMap),
		userMap: make(map[model.Code]string),
		mutex:   sync.Mutex{},
	}
}

func (s *Store) Read(key model.Code) (model.URL, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	value, ok := s.store[key]

	if !ok {
		return "", fmt.Errorf("key %s: %w", key, ErrNotFound)
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

	return nil
}

// InitializeWith инициализирует хранилище данными (без проверки на существование)
// Используется для массовой загрузки данных, например, из файла
func (s *Store) InitializeWith(data URLMap, userData map[model.Code]string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	maps.Copy(s.store, data)
	maps.Copy(s.userMap, userData)
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
	}

	return nil
}

// CreateOrGetURL создает новую запись или возвращает код существующей для данного URL
func (s *Store) CreateOrGetURL(code model.Code, url model.URL, userID string) (model.Code, bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Проверяем, существует ли уже такой URL
	for existingCode, existingURL := range s.store {
		if existingURL == url {
			// Обновляем userID для существующего кода
			s.userMap[existingCode] = userID
			return existingCode, false, nil // false = не создана новая запись
		}
	}

	// Проверяем, свободен ли код
	if _, exists := s.store[code]; exists {
		return "", false, fmt.Errorf("code %s: %w", code, ErrCodeAlreadyExists)
	}

	// Создаем новую запись
	s.store[code] = url
	s.userMap[code] = userID
	return code, true, nil // true = создана новая запись
}

// IsCodeUnique проверяет, свободен ли код в хранилище
func (s *Store) IsCodeUnique(code model.Code) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, exists := s.store[code]
	return !exists
}

// GetCodeByURL возвращает код для существующего URL
func (s *Store) GetCodeByURL(url model.URL) (model.Code, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for code, existingURL := range s.store {
		if existingURL == url {
			return code, nil
		}
	}

	return "", fmt.Errorf("URL not found: %w", ErrNotFound)
}

// GetURLsByUserID возвращает URL пользователя (не поддерживается в memory store)
func (s *Store) GetURLsByUserID(userID string, baseURL string) ([]model.UserURLResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var urls []model.UserURLResponse

	for code, storedUserID := range s.userMap {
		if storedUserID == userID {
			// Получаем оригинальный URL
			originalURL, exists := s.store[code]
			if !exists {
				continue // Несогласованность данных, пропускаем
			}

			// Формируем полный короткий URL
			shortURL, err := url.JoinPath(baseURL, string(code))
			if err != nil {
				continue // Пропускаем если не удалось сформировать URL
			}

			urls = append(urls, model.UserURLResponse{
				ShortURL:    shortURL,
				OriginalURL: string(originalURL),
			})
		}
	}

	return urls, nil
}
