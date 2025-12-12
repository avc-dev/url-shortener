package store

import (
	"context"
	"fmt"

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

	query := `
		SELECT original_url 
		FROM urls 
		WHERE code = $1
	`

	err := ds.pool.QueryRow(context.Background(), query, string(key)).Scan(&originalURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("key %s: %w", key, ErrNotFound)
		}
		return "", fmt.Errorf("failed to read from database: %w", err)
	}

	return model.URL(originalURL), nil
}

// Write сохраняет пару код-URL в базу данных
func (ds *DatabaseStore) Write(key model.Code, value model.URL) error {
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
		return fmt.Errorf("key %s: %w", key, ErrAlreadyExists)
	}

	// Вставляем новую запись
	query = `
		INSERT INTO urls (code, original_url) 
		VALUES ($1, $2)
	`

	_, err = ds.pool.Exec(ctx, query, string(key), string(value))
	if err != nil {
		return fmt.Errorf("failed to insert into database: %w", err)
	}

	return nil
}
