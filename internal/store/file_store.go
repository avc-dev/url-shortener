package store

import (
	"fmt"
	"net/url"

	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/google/uuid"
)

// FileStore декоратор над Store, который добавляет персистентность через файл
type FileStore struct {
	store       *Store
	fileStorage *FileStorage
	userMap     map[model.Code]string // code -> userID mapping
}

// NewFileStore создаёт FileStore и загружает данные из файла
func NewFileStore(filePath string) (*FileStore, error) {
	store := NewStore()
	fileStorage := NewFileStorage(filePath)

	fs := &FileStore{
		store:       store,
		fileStorage: fileStorage,
		userMap:     make(map[model.Code]string),
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
func (fs *FileStore) Write(key model.Code, value model.URL, userID string) error {
	if err := fs.store.Write(key, value, userID); err != nil {
		return fmt.Errorf("failed to write to in-memory store: %w", err)
	}

	fs.userMap[key] = userID

	// Добавляем только новую запись в файл (O(1) вместо O(n))
	entry := model.URLEntry{
		UUID:        uuid.New().String(),
		ShortURL:    string(key),
		OriginalURL: string(value),
		UserID:      userID,
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
		if entry.UserID != "" {
			fs.userMap[code] = entry.UserID
		}
	}

	fs.store.InitializeWith(data, fs.userMap)

	return nil
}

// WriteBatch записывает несколько значений в in-memory store и добавляет их в файл
func (fs *FileStore) WriteBatch(urls URLMap, userID string) error {
	// Сначала записываем в in-memory store
	if err := fs.store.WriteBatch(urls, userID); err != nil {
		return fmt.Errorf("failed to write batch to in-memory store: %w", err)
	}

	for code := range urls {
		fs.userMap[code] = userID
	}

	// Добавляем все записи в файл
	for code, url := range urls {
		entry := model.URLEntry{
			UUID:        uuid.New().String(),
			ShortURL:    string(code),
			OriginalURL: string(url),
			UserID:      userID,
		}

		if err := fs.fileStorage.Append(entry); err != nil {
			return fmt.Errorf("failed to append to file: %w", err)
		}
	}

	return nil
}

// IsCodeUnique проверяет, свободен ли код
func (fs *FileStore) IsCodeUnique(code model.Code) bool {
	return fs.store.IsCodeUnique(code)
}

// GetCodeByURL возвращает код для существующего URL
func (fs *FileStore) GetCodeByURL(url model.URL) (model.Code, error) {
	return fs.store.GetCodeByURL(url)
}

// CreateOrGetURL создает новую запись или возвращает код существующей для данного URL
func (fs *FileStore) CreateOrGetURL(code model.Code, url model.URL, userID string) (model.Code, bool, error) {
	// Для file storage мы всегда создаем новую запись (нет проверки дубликатов)
	// TODO: Можно добавить логику проверки существования URL для пользователя
	finalCode, created, err := fs.store.CreateOrGetURL(code, url, userID)
	if err != nil {
		return "", false, err
	}

	// Сохраняем user_id в памяти
	fs.userMap[finalCode] = userID

	// Сохраняем в файл
	entry := model.URLEntry{
		UUID:        uuid.New().String(),
		ShortURL:    string(finalCode),
		OriginalURL: string(url),
		UserID:      userID,
	}

	if err := fs.fileStorage.Append(entry); err != nil {
		return "", false, fmt.Errorf("failed to append to file: %w", err)
	}

	return finalCode, created, nil
}

// GetURLsByUserID возвращает все URL для указанного пользователя из file store
func (fs *FileStore) GetURLsByUserID(userID string, baseURL string) ([]model.UserURLResponse, error) {
	var urls []model.UserURLResponse

	for code, storedUserID := range fs.userMap {
		if storedUserID == userID {
			// Получаем оригинальный URL
			originalURL, err := fs.store.Read(code)
			if err != nil {
				continue // Пропускаем если URL не найден (несогласованность данных)
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
