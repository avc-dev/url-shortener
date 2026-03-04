package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// FileObserver записывает события аудита в файл (по одному JSON-объекту на строку)
type FileObserver struct {
	path string
	mu   sync.Mutex
}

// NewFileObserver создает FileObserver, который пишет аудит в файл по указанному пути
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

// Notify сериализует событие в JSON и дописывает его в конец файла
func (f *FileObserver) Notify(_ context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open audit file: %w", err)
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%s\n", data)
	return err
}
