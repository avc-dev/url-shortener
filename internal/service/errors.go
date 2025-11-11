package service

import "errors"

var (
	// ErrMaxRetriesExceeded возвращается когда не удалось сгенерировать уникальный код
	// после максимального количества попыток
	ErrMaxRetriesExceeded = errors.New("max retries exceeded for code generation")
)

