package config

import (
	"fmt"
	"strconv"
	"strings"
)

// NetworkAddress представляет сетевой адрес в формате "host:port".
// Реализует интерфейсы flag.Value и encoding.TextUnmarshaler для удобной
// загрузки из флагов командной строки и переменных окружения.
type NetworkAddress struct {
	Host string
	Port int
}

// String возвращает адрес в формате "host:port".
func (a NetworkAddress) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

// Set разбирает строку "host:port" и заполняет поля структуры.
// Возвращает ошибку, если формат некорректен или порт не является числом.
func (a *NetworkAddress) Set(value string) error {
	parts := strings.Split(value, ":")

	if len(parts) != 2 {
		return fmt.Errorf("invalid network address format: %s", value)
	}

	a.Host = parts[0]

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}
	a.Port = port

	return nil
}

// UnmarshalText реализует encoding.TextUnmarshaler, делегируя парсинг методу Set.
func (a *NetworkAddress) UnmarshalText(text []byte) error {
	return a.Set(string(text))
}
