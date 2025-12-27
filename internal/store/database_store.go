package store

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/avc-dev/url-shortener/internal/config/db"
	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseStore реализует Store интерфейс для PostgreSQL
type DatabaseStore struct {
	pool *pgxpool.Pool
}

// NewDatabaseStore создает новый DatabaseStore
func NewDatabaseStore(database db.Database) *DatabaseStore {
	// Получаем pgxpool.Pool из адаптера
	adapter, ok := database.(*db.DBAdapter)
	if !ok {
		panic("DatabaseStore requires DBAdapter")
	}

	return &DatabaseStore{
		pool: adapter.Pool,
	}
}

// Read читает оригинальный URL по короткому коду
func (ds *DatabaseStore) Read(key model.Code) (model.URL, error) {
	var originalURL string
	var isDeleted bool

	query := `
		SELECT original_url, is_deleted
		FROM urls
		WHERE code = $1
	`

	err := ds.pool.QueryRow(context.Background(), query, string(key)).Scan(&originalURL, &isDeleted)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("key %s: %w", key, ErrNotFound)
		}
		return "", fmt.Errorf("failed to read from database: %w", err)
	}

	if isDeleted {
		return "", fmt.Errorf("key %s: %w", key, ErrURLDeleted)
	}

	return model.URL(originalURL), nil
}

// Write сохраняет пару код-URL с userID в базу данных
func (ds *DatabaseStore) Write(key model.Code, value model.URL, userID string) error {
	ctx := context.Background()

	// Проверяем существование ключа
	var exists bool

	query := `
		SELECT EXISTS
		(SELECT 1 FROM urls WHERE code = $1)
	`

	err := ds.pool.QueryRow(ctx, query, string(key)).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check key existence: %w", err)
	}

	if exists {
		return fmt.Errorf("code %s: %w", key, ErrCodeAlreadyExists)
	}

	// Вставляем новую запись
	query = `
		INSERT INTO urls (code, original_url, user_id)
		VALUES ($1, $2, $3)
	`

	_, err = ds.pool.Exec(ctx, query, string(key), string(value), userID)
	if err != nil {
		return fmt.Errorf("failed to insert into database: %w", err)
	}

	return nil
}

// WriteBatch сохраняет несколько пар код-URL с userID в базу данных в рамках одной транзакции
func (ds *DatabaseStore) WriteBatch(urls map[model.Code]model.URL, userID string) error {
	ctx := context.Background()

	// Начинаем транзакцию
	tx, err := ds.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // откатим транзакцию в случае ошибки

	// Проверяем существование всех кодов перед вставкой
	codes := make([]string, 0, len(urls))
	args := make([]interface{}, 0, len(urls))
	for code := range urls {
		codes = append(codes, string(code))
		args = append(args, string(code))
	}

	if len(codes) > 0 {
		// Создаем плейсхолдеры для IN запроса
		placeholders := ""
		for i := range codes {
			if i > 0 {
				placeholders += ","
			}
			placeholders += fmt.Sprintf("$%d", i+1)
		}

		query := fmt.Sprintf(`
			SELECT code FROM urls WHERE code IN (%s)
		`, placeholders)

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to check existing codes: %w", err)
		}
		defer rows.Close()

		// Если найдены существующие коды, возвращаем ошибку
		if rows.Next() {
			var existingCode string
			if err := rows.Scan(&existingCode); err != nil {
				return fmt.Errorf("failed to scan existing code: %w", err)
			}
			return fmt.Errorf("code %s: %w", existingCode, ErrCodeAlreadyExists)
		}
		rows.Close()
	}

	// Вставляем все записи
	query := `
		INSERT INTO urls (code, original_url, user_id)
		VALUES ($1, $2, $3)
	`

	for code, url := range urls {
		_, err = tx.Exec(ctx, query, string(code), string(url), userID)
		if err != nil {
			return fmt.Errorf("failed to insert into database: %w", err)
		}
	}

	// Фиксируем транзакцию
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreateOrGetURL создает новую запись или возвращает код существующей для данного URL
// Использует CTE для атомарной проверки существования и вставки без изменения существующего кода
func (ds *DatabaseStore) CreateOrGetURL(code model.Code, url model.URL, userID string) (model.Code, bool, error) {
	ctx := context.Background()

	// Используем CTE для атомарной проверки существования URL и вставки
	query := `
		WITH existing_url AS (
			SELECT code FROM urls WHERE original_url = $2 AND user_id = $3
		),
		insert_result AS (
			INSERT INTO urls (code, original_url, user_id)
			SELECT $1, $2, $3
			WHERE NOT EXISTS (SELECT 1 FROM existing_url)
			RETURNING code
		)
		SELECT
			COALESCE(existing.code, inserted.code) as final_code,
			CASE
				WHEN existing.code IS NOT NULL THEN false
				WHEN inserted.code IS NOT NULL THEN true
				ELSE false
			END as created
		FROM (SELECT 1) dummy
		LEFT JOIN existing_url existing ON true
		LEFT JOIN insert_result inserted ON true
	`

	var finalCode string
	var created bool

	err := ds.pool.QueryRow(ctx, query, string(code), string(url), userID).Scan(&finalCode, &created)
	if err != nil {
		return "", false, fmt.Errorf("failed to create or get URL: %w", err)
	}

	return model.Code(finalCode), created, nil
}

// IsCodeUnique проверяет, свободен ли код в базе данных
func (ds *DatabaseStore) IsCodeUnique(code model.Code) bool {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE code = $1)`

	err := ds.pool.QueryRow(context.Background(), query, string(code)).Scan(&exists)
	if err != nil {
		// В случае ошибки считаем код занятым для безопасности
		return false
	}

	return !exists
}

// GetCodeByURL возвращает код для существующего URL
func (ds *DatabaseStore) GetCodeByURL(url model.URL) (model.Code, error) {
	var code string
	query := `SELECT code FROM urls WHERE original_url = $1`

	err := ds.pool.QueryRow(context.Background(), query, string(url)).Scan(&code)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("URL not found: %w", ErrNotFound)
		}
		return "", fmt.Errorf("failed to get code by URL: %w", err)
	}

	return model.Code(code), nil
}

// GetURLsByUserID возвращает все URL для указанного пользователя (исключая удалённые)
func (ds *DatabaseStore) GetURLsByUserID(userID string, baseURL string) ([]model.UserURLResponse, error) {
	ctx := context.Background()

	query := `
		SELECT code, original_url
		FROM urls
		WHERE user_id = $1 AND is_deleted = false
		ORDER BY created_at DESC
	`

	rows, err := ds.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query URLs by user ID: %w", err)
	}
	defer rows.Close()

	var urls []model.UserURLResponse
	for rows.Next() {
		var code, originalURL string
		if err := rows.Scan(&code, &originalURL); err != nil {
			return nil, fmt.Errorf("failed to scan URL row: %w", err)
		}

		shortURL, err := url.JoinPath(baseURL, code)
		if err != nil {
			return nil, fmt.Errorf("failed to construct short URL: %w", err)
		}

		urls = append(urls, model.UserURLResponse{
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over URL rows: %w", err)
	}

	return urls, nil
}

// IsURLOwnedByUser проверяет, принадлежит ли URL указанному пользователю
func (ds *DatabaseStore) IsURLOwnedByUser(code model.Code, userID string) bool {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM urls WHERE code = $1 AND user_id = $2 AND is_deleted = false)`
	err := ds.pool.QueryRow(context.Background(), query, string(code), userID).Scan(&exists)
	return err == nil && exists
}

// DeleteURLsBatch помечает несколько URL как удалённые для указанного пользователя
// Выполняет batch update без дополнительной валидации (валидация должна происходить на более высоком уровне)
func (ds *DatabaseStore) DeleteURLsBatch(codes []model.Code, userID string) error {
	if len(codes) == 0 {
		return nil
	}

	ctx := context.Background()
	return ds.batchUpdateDeletedFlag(ctx, codes, true)
}

// batchUpdateDeletedFlag выполняет batch update флага is_deleted
func (ds *DatabaseStore) batchUpdateDeletedFlag(ctx context.Context, codes []model.Code, isDeleted bool) error {
	// Создаем placeholders для IN запроса
	placeholders := make([]string, len(codes))
	args := make([]interface{}, len(codes)+1)

	for i, code := range codes {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = string(code)
	}
	args[len(codes)] = isDeleted

	query := fmt.Sprintf(`
		UPDATE urls
		SET is_deleted = $%d
		WHERE code IN (%s)
	`, len(codes)+1, fmt.Sprintf("(%s)", strings.Join(placeholders, ",")))

	_, err := ds.pool.Exec(ctx, query, args...)
	return err
}
