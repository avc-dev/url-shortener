// Package config отвечает за загрузку и хранение конфигурации приложения.
// Параметры считываются из флагов командной строки, а затем переопределяются
// значениями переменных окружения (ENV имеет высший приоритет).
package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

// RetryConfig хранит параметры повторных попыток генерации кода.
type RetryConfig struct {
	// MaxAttempts — максимальное число попыток генерации уникального короткого кода.
	MaxAttempts int `env:"MAX_ATTEMPTS" envDefault:"100"`
}

// Config содержит всю конфигурацию приложения.
// Поля помечены тегами env для автоматической загрузки из переменных окружения.
type Config struct {
	// ServerAddress — адрес и порт HTTP-сервера (флаг -a / SERVER_ADDRESS).
	ServerAddress NetworkAddress `env:"SERVER_ADDRESS"`
	// BaseURL — базовый URL для формирования коротких ссылок (флаг -b / BASE_URL).
	BaseURL URLPrefix `env:"BASE_URL"`
	// FileStoragePath — путь к файлу для хранения данных (флаг -f / FILE_STORAGE_PATH).
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	// DatabaseDSN — строка подключения к PostgreSQL (флаг -d / DATABASE_DSN).
	DatabaseDSN string `env:"DATABASE_DSN"`
	// JWTSecret — секрет для подписи JWT-токенов (флаг -j / JWT_SECRET).
	JWTSecret string `env:"JWT_SECRET" envDefault:"your-secret-key"`
	// AuditFile — путь к файлу аудита (флаг --audit-file / AUDIT_FILE).
	AuditFile string `env:"AUDIT_FILE"`
	// AuditURL — URL удалённого сервера аудита (флаг --audit-url / AUDIT_URL).
	AuditURL string `env:"AUDIT_URL"`
	// Retry — конфигурация повторных попыток с префиксом RETRY_.
	Retry RetryConfig `envPrefix:"RETRY_"`
}

// NewDefaultConfig возвращает конфигурацию со значениями по умолчанию
func NewDefaultConfig() *Config {
	return &Config{
		ServerAddress: NetworkAddress{Host: "localhost", Port: 8080},
		BaseURL:       URLPrefix("http://localhost:8080/"),
		JWTSecret:     "your-secret-key",
		Retry:         RetryConfig{MaxAttempts: 100},
	}
}

// Load загружает конфигурацию с учётом приоритетов:
// 1. ENV переменные (высший приоритет)
// 2. Флаги командной строки
// 3. Значения по умолчанию (низший приоритет)
func Load() (*Config, error) {
	cfg := NewDefaultConfig()

	addrFlag := flag.String("a", "", "address to run HTTP server")
	baseURLFlag := flag.String("b", "", "base URL for shortened URL")
	fileStoragePathFlag := flag.String("f", "", "file storage path")
	databaseDSNFlag := flag.String("d", "", "database DSN")
	jwtSecretFlag := flag.String("j", "", "JWT secret key")
	maxAttemptsFlag := flag.Int("r", 0, "maximum attempts for code generation")
	auditFileFlag := flag.String("audit-file", "", "path to audit log file")
	auditURLFlag := flag.String("audit-url", "", "URL of remote audit server")
	flag.Parse()

	if *addrFlag != "" {
		if err := cfg.ServerAddress.Set(*addrFlag); err != nil {
			return nil, fmt.Errorf("invalid server address flag: %w", err)
		}
	}
	if *baseURLFlag != "" {
		if err := cfg.BaseURL.Set(*baseURLFlag); err != nil {
			return nil, fmt.Errorf("invalid base URL flag: %w", err)
		}
	}
	if *fileStoragePathFlag != "" {
		cfg.FileStoragePath = *fileStoragePathFlag
	}
	if *databaseDSNFlag != "" {
		cfg.DatabaseDSN = *databaseDSNFlag
	}
	if *jwtSecretFlag != "" {
		cfg.JWTSecret = *jwtSecretFlag
	}
	if *maxAttemptsFlag > 0 {
		cfg.Retry.MaxAttempts = *maxAttemptsFlag
	}
	if *auditFileFlag != "" {
		cfg.AuditFile = *auditFileFlag
	}
	if *auditURLFlag != "" {
		cfg.AuditURL = *auditURLFlag
	}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	return cfg, nil
}
