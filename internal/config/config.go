// Package config отвечает за загрузку и хранение конфигурации приложения.
// Параметры считываются из флагов командной строки, а затем переопределяются
// значениями переменных окружения (ENV имеет высший приоритет).
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
)

// RetryConfig хранит параметры повторных попыток генерации кода.
type RetryConfig struct {
	// MaxAttempts — максимальное число попыток генерации уникального короткого кода.
	MaxAttempts int `env:"MAX_ATTEMPTS" envDefault:"100" json:"retry_max_attempts"`
}

// Config содержит всю конфигурацию приложения.
// Поля помечены тегами env для автоматической загрузки из переменных окружения
// и тегами json для загрузки из файла конфигурации.
type Config struct {
	BaseURL         URLPrefix      `env:"BASE_URL"           json:"base_url"`
	FileStoragePath string         `env:"FILE_STORAGE_PATH"  json:"file_storage_path"`
	DatabaseDSN     string         `env:"DATABASE_DSN"       json:"database_dsn"`
	JWTSecret       string         `env:"JWT_SECRET" envDefault:"your-secret-key" json:"jwt_secret"`
	AuditFile       string         `env:"AUDIT_FILE"         json:"audit_file"`
	AuditURL        string         `env:"AUDIT_URL"          json:"audit_url"`
	TrustedSubnet   string         `env:"TRUSTED_SUBNET"     json:"trusted_subnet"`
	ServerAddress   NetworkAddress `env:"SERVER_ADDRESS"     json:"server_address"`
	Retry           RetryConfig    `envPrefix:"RETRY_"       json:"retry"`
	EnableHTTPS     bool           `env:"ENABLE_HTTPS"       json:"enable_https"`
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
// 3. JSON-файл конфигурации (-c / -config / CONFIG)
// 4. Значения по умолчанию (низший приоритет)
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
	trustedSubnetFlag := flag.String("t", "", "trusted subnet in CIDR notation (e.g. 192.168.1.0/24)")
	enableHTTPSFlag := flag.Bool("s", false, "enable HTTPS")
	configFileFlag := flag.String("c", "", "path to JSON config file")
	flag.StringVar(configFileFlag, "config", "", "path to JSON config file")
	flag.Parse()

	// Определяем путь к файлу конфигурации: CONFIG env > -c/-config флаг.
	// os.Getenv используется намеренно — env.Parse() ещё не запускался.
	configFilePath := os.Getenv("CONFIG")
	if configFilePath == "" {
		configFilePath = *configFileFlag
	}

	// Применяем JSON-файл конфигурации (низший приоритет после флагов и ENV).
	// Десериализуем прямо в cfg: отсутствующие поля не трогают уже заданные дефолты.
	if configFilePath != "" {
		data, err := os.ReadFile(configFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Флаги переопределяют значения из файла конфигурации.
	if *enableHTTPSFlag {
		cfg.EnableHTTPS = true
	}
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
	if *trustedSubnetFlag != "" {
		cfg.TrustedSubnet = *trustedSubnetFlag
	}

	// ENV переменные имеют высший приоритет.
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	return cfg, nil
}
