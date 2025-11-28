package store

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/avc-dev/url-shortener/internal/model"
)

// FileStorage управляет персистентным хранилищем URL в JSON файле
type FileStorage struct {
	filePath string
}

// NewFileStorage создаёт новый FileStorage
func NewFileStorage(filePath string) *FileStorage {
	return &FileStorage{
		filePath: filePath,
	}
}

// Load загружает все записи из файла
func (fs *FileStorage) Load() ([]model.URLEntry, error) {
	if _, err := os.Stat(fs.filePath); os.IsNotExist(err) {
		return []model.URLEntry{}, nil
	}

	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) == 0 {
		return []model.URLEntry{}, nil
	}

	var entries []model.URLEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return entries, nil
}

// Save сохраняет все записи в файл
func (fs *FileStorage) Save(entries []model.URLEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(fs.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

