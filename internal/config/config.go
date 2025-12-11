package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ServerAddress   NetworkAddress `env:"SERVER_ADDRESS"`
	BaseURL         URLPrefix      `env:"BASE_URL"`
	FileStoragePath string         `env:"FILE_STORAGE_PATH"`
	DatabaseDSN     string         `env:"DATABASE_DSN"`
}

// NewDefaultConfig возвращает конфигурацию со значениями по умолчанию
func NewDefaultConfig() *Config {
	return &Config{
		ServerAddress: NetworkAddress{Host: "localhost", Port: 8080},
		BaseURL:       URLPrefix("http://localhost:8080/"),
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

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	return cfg, nil
}
