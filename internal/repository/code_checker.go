package repository

import (
	"errors"
	"fmt"

	"github.com/avc-dev/url-shortener/internal/model"
	"github.com/avc-dev/url-shortener/internal/store"
)

// Exists проверяет существование кода в хранилище
// Возвращает true если код существует, false если код свободен
// Возвращает ошибку только в случае проблем с хранилищем (не "not found")
func (r *Repository) Exists(code model.Code) (bool, error) {
	_, err := r.underlying.Read(code)
	if err != nil {
		// Если ошибка - "not found", значит код свободен
		if errors.Is(err, store.ErrNotFound) {
			return false, nil
		}
		// Любая другая ошибка - проблема с хранилищем
		return false, fmt.Errorf("failed to check code existence: %w", err)
	}

	// Код существует
	return true, nil
}
