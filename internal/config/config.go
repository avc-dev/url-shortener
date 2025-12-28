package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type RetryConfig struct {
	MaxAttempts int `env:"MAX_ATTEMPTS" envDefault:"100"`
}

type Config struct {
	ServerAddress   NetworkAddress `env:"SERVER_ADDRESS"`
	BaseURL         URLPrefix      `env:"BASE_URL"`
	FileStoragePath string         `env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string         `env:"DATABASE_DSN"`
	JWTSecret       string         `env:"JWT_SECRET" envDefault:"your-secret-key"`
	Retry           RetryConfig    `envPrefix:"RETRY_"`
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

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	return cfg, nil
}
