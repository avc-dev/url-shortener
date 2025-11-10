package store

import (
	"errors"
	"fmt"
	"sync"

	"github.com/avc-dev/url-shortener/internal/model"
)

var (
	ErrNotFound      = errors.New("key not found")
	ErrAlreadyExists = errors.New("key already exists")
)

type Store struct {
	store map[model.Code]model.URL
	mutex sync.Mutex
}

func NewStore() *Store {
	return &Store{
		store: make(map[model.Code]model.URL),
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
