package config

import (
	"fmt"
	"strings"
)

// URLPrefix — строковый тип для базового URL сервиса (например, "http://localhost:8080/").
// Реализует интерфейсы flag.Value и encoding.TextUnmarshaler.
type URLPrefix string

// String возвращает строковое представление префикса.
func (p URLPrefix) String() string {
	return string(p)
}

// Set проверяет, что значение начинается с "http", и сохраняет его.
// Возвращает ошибку для невалидных схем.
func (p *URLPrefix) Set(value string) error {
	if !strings.HasPrefix(value, "http") {
		return fmt.Errorf("invalid URL prefix format: %s", value)
	}

	*p = URLPrefix(value)

	return nil
}

// UnmarshalText реализует encoding.TextUnmarshaler, делегируя парсинг методу Set.
func (p *URLPrefix) UnmarshalText(text []byte) error {
	return p.Set(string(text))
}
