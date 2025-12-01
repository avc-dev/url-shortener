package store

import (
	"bufio"
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

// Load загружает все записи из файла (JSONL формат - каждая запись на отдельной строке)
func (fs *FileStorage) Load() ([]model.URLEntry, error) {
	file, err := os.Open(fs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.URLEntry{}, nil
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var entries []model.URLEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry model.URLEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON line: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return entries, nil
}

// Save сохраняет все записи в файл (JSONL формат - каждая запись на отдельной строке)
// Используется для компакции или начального сохранения
func (fs *FileStorage) Save(entries []model.URLEntry) error {
	file, err := os.Create(fs.filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			return fmt.Errorf("failed to encode entry: %w", err)
		}
	}

	return nil
}

// Append добавляет одну запись в конец файла (JSONL формат)
func (fs *FileStorage) Append(entry model.URLEntry) error {
	file, err := os.OpenFile(fs.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for append: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to encode entry: %w", err)
	}

	return nil
}
