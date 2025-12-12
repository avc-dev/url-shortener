package store

import (
	"maps"
	"errors"
	"fmt"
	"sync"

	"github.com/avc-dev/url-shortener/internal/model"
)

var (
	ErrNotFound      = errors.New("key not found")
	ErrAlreadyExists = errors.New("key already exists")
)

// URLMap представляет маппинг коротких кодов на оригинальные URL
type URLMap = map[model.Code]model.URL

type Store struct {
	store URLMap
	mutex sync.Mutex
}

func NewStore() *Store {
	return &Store{
		store: make(URLMap),
		mutex: sync.Mutex{},
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

func (s *Store) Write(key model.Code, value model.URL) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Проверяем существование ключа напрямую, без вызова Read (чтобы избежать deadlock)
	if _, exists := s.store[key]; exists {
		return fmt.Errorf("key %s: %w", key, ErrAlreadyExists)
	}

	s.store[key] = value

	return nil
}

// InitializeWith инициализирует хранилище данными (без проверки на существование)
// Используется для массовой загрузки данных, например, из файла
func (s *Store) InitializeWith(data URLMap) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	maps.Copy(s.store, data)
}

// WriteBatch сохраняет несколько пар код-URL в хранилище атомарно
func (s *Store) WriteBatch(urls URLMap) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Проверяем существование всех кодов перед вставкой
	for code := range urls {
		if _, exists := s.store[code]; exists {
			return fmt.Errorf("key %s: %w", code, ErrAlreadyExists)
		}
	}

	// Вставляем все записи
	for code, url := range urls {
		s.store[code] = url
	}

	return nil
}
