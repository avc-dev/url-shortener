package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // Регистрируем pgx драйвер для database/sql
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

	// Создаем sql.DB с pgx драйвером для миграций
	sqlDB, err := sql.Open("pgx", c.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open sql database: %w", err)
	}

	// Настраиваем соединение
	sqlDB.SetMaxOpenConns(int(c.MaxConns))
	sqlDB.SetMaxIdleConns(int(c.MinConns))
	sqlDB.SetConnMaxLifetime(c.MaxConnLifetime)
	sqlDB.SetConnMaxIdleTime(c.MaxConnIdleTime)

	// Проверяем подключение
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	config, err := pgxpool.ParseConfig(c.DSN)
	if err != nil {
		sqlDB.Close()
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
		sqlDB.Close()
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверяем подключение
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return NewDBAdapter(pool, sqlDB), nil
}

//go:generate mockery --name Database

// Database интерфейс для работы с базой данных
type Database interface {
	Ping(ctx context.Context) error
	Close()
	// Возвращает *sql.DB для миграций и других операций
	DB() *sql.DB
}

// Ping проверяет подключение к базе данных
func Ping(ctx context.Context, db Database) error {
	return db.Ping(ctx)
}

// DBAdapter адаптер для pgxpool.Pool к Database интерфейсу
type DBAdapter struct {
	Pool  *pgxpool.Pool
	SQLDB *sql.DB
}

// NewDBAdapter создает новый адаптер
func NewDBAdapter(pool *pgxpool.Pool, sqlDB *sql.DB) *DBAdapter {
	return &DBAdapter{
		Pool:  pool,
		SQLDB: sqlDB,
	}
}

// Ping проверяет подключение
func (d *DBAdapter) Ping(ctx context.Context) error {
	return d.Pool.Ping(ctx)
}

// Close закрывает соединения
func (d *DBAdapter) Close() {
	d.Pool.Close()
	if d.SQLDB != nil {
		d.SQLDB.Close()
	}
}

// DB возвращает *sql.DB
func (d *DBAdapter) DB() *sql.DB {
	return d.SQLDB
}
