package service

import "github.com/avc-dev/url-shortener/internal/model"

// CodeChecker проверяет существование кода в хранилище
type CodeChecker interface {
	// Exists проверяет, существует ли код в системе
	// Возвращает true если код уже существует, false если свободен
	Exists(code model.Code) (bool, error)
}

