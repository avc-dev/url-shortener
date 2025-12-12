package store

import (
	"errors"
	"fmt"
	"maps"
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
		return fmt.Errorf("code %s: %w", key, ErrCodeAlreadyExists)
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
			return fmt.Errorf("code %s: %w", code, ErrCodeAlreadyExists)
		}
	}

	// Вставляем все записи
	for code, url := range urls {
		s.store[code] = url
	}

	return nil
}

// CreateOrGetCode создает новый код для URL или возвращает существующий если URL уже есть в хранилище
func (s *Store) CreateOrGetCode(value model.URL) (model.Code, bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Проверяем, существует ли уже такой URL
	for existingCode, existingURL := range s.store {
		if existingURL == value {
			return existingCode, false, nil // false = не создана новая запись
		}
	}

	// Если URL не существует, создаем новый код
	for {
		code := model.Code(randomString())
		if _, exists := s.store[code]; !exists {
			s.store[code] = value
			return code, true, nil // true = создана новая запись
		}
	}
}
