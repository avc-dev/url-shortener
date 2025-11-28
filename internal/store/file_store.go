package store

import (
	"fmt"

	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/google/uuid"
)

// FileStore декоратор над Store, который добавляет персистентность через файл
type FileStore struct {
	store       *Store
	fileStorage *FileStorage
}

// NewFileStore создаёт FileStore и загружает данные из файла
func NewFileStore(filePath string) (*FileStore, error) {
	store := NewStore()
	fileStorage := NewFileStorage(filePath)

	fs := &FileStore{
		store:       store,
		fileStorage: fileStorage,
	}

	// Загружаем данные из файла при инициализации
	if err := fs.loadFromFile(); err != nil {
		return nil, fmt.Errorf("failed to load data from file: %w", err)
	}

	return fs, nil
}

// Read читает значение из in-memory store
func (fs *FileStore) Read(key model.Code) (model.URL, error) {
	return fs.store.Read(key)
}

// Write записывает значение в in-memory store и сохраняет в файл
func (fs *FileStore) Write(key model.Code, value model.URL) error {
	if err := fs.store.Write(key, value); err != nil {
		return fmt.Errorf("failed to write to in-memory store: %w", err)
	}

	if err := fs.saveToFile(); err != nil {
		return fmt.Errorf("failed to save to file: %w", err)
	}

	return nil
}

// loadFromFile загружает данные из файла в in-memory store
func (fs *FileStore) loadFromFile() error {
	entries, err := fs.fileStorage.Load()
	if err != nil {
		return fmt.Errorf("failed to load data from file: %w", err)
	}

	// Загружаем каждую запись в in-memory store
	for _, entry := range entries {
		code := model.Code(entry.ShortURL)
		url := model.URL(entry.OriginalURL)

		// Используем прямую запись в map, чтобы избежать проверки на существование
		fs.store.mutex.Lock()
		fs.store.store[code] = url
		fs.store.mutex.Unlock()
	}

	return nil
}

// saveToFile сохраняет все данные из in-memory store в файл
func (fs *FileStore) saveToFile() error {
	fs.store.mutex.Lock()
	defer fs.store.mutex.Unlock()

	entries := make([]model.URLEntry, 0, len(fs.store.store))
	for code, url := range fs.store.store {
		entries = append(entries, model.URLEntry{
			UUID:        uuid.New().String(),
			ShortURL:    string(code),
			OriginalURL: string(url),
		})
	}

	return fs.fileStorage.Save(entries)
}

