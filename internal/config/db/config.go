package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config содержит настройки подключения к базе данных
type Config struct {
	DSN               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// NewConfig создает конфигурацию подключения к БД
func NewConfig(dsn string) *Config {
	return &Config{
		DSN:               dsn,
		MaxConns:          10,
		MinConns:          1,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   time.Minute * 30,
		HealthCheckPeriod: time.Minute,
	}
}

// Connect создает пул подключений к PostgreSQL
func (c *Config) Connect(ctx context.Context) (Database, error) {
	if c.DSN == "" {
		return nil, fmt.Errorf("database DSN is required")
	}

	config, err := pgxpool.ParseConfig(c.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Настраиваем пул подключений
	config.MaxConns = c.MaxConns
	config.MinConns = c.MinConns
	config.MaxConnLifetime = c.MaxConnLifetime
	config.MaxConnIdleTime = c.MaxConnIdleTime
	config.HealthCheckPeriod = c.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверяем подключение
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

//go:generate mockery --name Database

// Database интерфейс для работы с базой данных
type Database interface {
	Ping(ctx context.Context) error
	Close()
}

// Ping проверяет подключение к базе данных
func Ping(ctx context.Context, db Database) error {
	return db.Ping(ctx)
}
