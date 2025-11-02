package store

import (
	"fmt"
	"sync"

	"github.com/avc-dev/url-shortener/internal/model"
)

type Store struct {
	store map[model.Code]model.URL
	mutex sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		store: make(map[model.Code]model.URL),
		mutex: sync.RWMutex{},
	}
}

func (s *Store) Read(key model.Code) (model.URL, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	value, ok := s.store[key]

	if !ok {
		return "", fmt.Errorf("key %s not exist", key)
	}

	return value, nil
}

func (s *Store) Write(key model.Code, value model.URL) error {
	_, err := s.Read(key)
	if err == nil {
		return fmt.Errorf("key %s already exists", key)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.store[key] = value

	return nil
}
