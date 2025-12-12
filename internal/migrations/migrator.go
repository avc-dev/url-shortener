package migrations

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
)

//go:embed schema/*.sql
var migrationFiles embed.FS

// Migrator управляет миграциями базы данных
type Migrator struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewMigrator создает новый экземпляр migrator
func NewMigrator(db *sql.DB, logger *zap.Logger) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger,
	}
}

// RunUp применяет все миграции вверх
func (m *Migrator) RunUp() error {
	m.logger.Info("Starting database migrations")

	// Создаем источник миграций из embed файлов
	source, err := iofs.New(migrationFiles, "schema")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Создаем драйвер для PostgreSQL
	driver, err := postgres.WithInstance(m.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	// Создаем экземпляр migrate
	migrateInstance, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer migrateInstance.Close()

	// Применяем все миграции
	err = migrateInstance.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		m.logger.Info("No migrations to apply")
	} else {
		m.logger.Info("Migrations applied successfully")
	}

	return nil
}

// GetVersion возвращает текущую версию миграций
func (m *Migrator) GetVersion() (uint, bool, error) {
	source, err := iofs.New(migrationFiles, "schema")
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration source: %w", err)
	}

	driver, err := postgres.WithInstance(m.db, &postgres.Config{})
	if err != nil {
		return 0, false, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	migrateInstance, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer migrateInstance.Close()

	version, dirty, err := migrateInstance.Version()
	return version, dirty, err
}
