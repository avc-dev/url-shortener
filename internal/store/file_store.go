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

// Write записывает значение в in-memory store и добавляет в файл
func (fs *FileStore) Write(key model.Code, value model.URL) error {
	if err := fs.store.Write(key, value); err != nil {
		return fmt.Errorf("failed to write to in-memory store: %w", err)
	}

	// Добавляем только новую запись в файл (O(1) вместо O(n))
	entry := model.URLEntry{
		UUID:        uuid.New().String(),
		ShortURL:    string(key),
		OriginalURL: string(value),
	}

	if err := fs.fileStorage.Append(entry); err != nil {
		return fmt.Errorf("failed to append to file: %w", err)
	}

	return nil
}

// loadFromFile загружает данные из файла в in-memory store
func (fs *FileStore) loadFromFile() error {
	entries, err := fs.fileStorage.Load()
	if err != nil {
		return fmt.Errorf("failed to load data from file: %w", err)
	}

	data := make(URLMap, len(entries))
	for _, entry := range entries {
		code := model.Code(entry.ShortURL)
		url := model.URL(entry.OriginalURL)
		data[code] = url
	}

	fs.store.InitializeWith(data)

	return nil
}

// WriteBatch записывает несколько значений в in-memory store и добавляет их в файл
func (fs *FileStore) WriteBatch(urls URLMap) error {
	// Сначала записываем в in-memory store
	if err := fs.store.WriteBatch(urls); err != nil {
		return fmt.Errorf("failed to write batch to in-memory store: %w", err)
	}

	// Добавляем все записи в файл
	for code, url := range urls {
		entry := model.URLEntry{
			UUID:        uuid.New().String(),
			ShortURL:    string(code),
			OriginalURL: string(url),
		}

		if err := fs.fileStorage.Append(entry); err != nil {
			return fmt.Errorf("failed to append to file: %w", err)
		}
	}

	return nil
}

// CreateOrGetCode создает новый код для URL или возвращает существующий
func (fs *FileStore) CreateOrGetCode(value model.URL) (model.Code, bool, error) {
	code, created, err := fs.store.CreateOrGetCode(value)
	if err != nil {
		return "", false, err
	}

	// Если создана новая запись, сохраняем в файл
	if created {
		entry := model.URLEntry{
			UUID:        uuid.New().String(),
			ShortURL:    string(code),
			OriginalURL: string(value),
		}

		if err := fs.fileStorage.Append(entry); err != nil {
			return "", false, fmt.Errorf("failed to append to file: %w", err)
		}
	}

	return code, created, nil
}
