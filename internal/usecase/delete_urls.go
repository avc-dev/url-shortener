package usecase

import (
	"github.com/avc-dev/url-shortener/internal/model"
	"go.uber.org/zap"
)

// DeleteURLs удаляет несколько URL для указанного пользователя
// Использует асинхронную обработку с воркерами и fanIn паттерном для валидации
func (u *URLUsecase) DeleteURLs(codes []string, userID string) error {
	// Конвертируем строки в model.Code
	modelCodes := make([]model.Code, len(codes))
	for i, code := range codes {
		modelCodes[i] = model.Code(code)
	}

	// Выполняем асинхронное удаление с воркерами
	go u.deleteURLsAsync(modelCodes, userID, codes)

	return nil
}

// deleteURLsAsync асинхронно удаляет URL с использованием воркеров и fanIn паттерна
func (u *URLUsecase) deleteURLsAsync(codes []model.Code, userID string, originalCodes []string) {
	// Создаем валидатор для проверки принадлежности URL пользователю
	validator := func(code model.Code) bool {
		return u.repo.IsURLOwnedByUser(code, userID)
	}

	// Создаем процессор для batch удаления
	processor := func(validCodes []model.Code) {
		err := u.repo.DeleteURLsBatch(validCodes, userID)
		if err != nil {
			u.logger.Error("failed to delete URLs batch",
				zap.Strings("codes", originalCodes),
				zap.String("userID", userID),
				zap.Int("validCodesCount", len(validCodes)),
				zap.Error(err),
			)
		} else {
			u.logger.Info("successfully deleted URLs batch",
				zap.Strings("codes", originalCodes),
				zap.String("userID", userID),
				zap.Int("validCodesCount", len(validCodes)),
			)
		}
	}

	// Выполняем асинхронную обработку с воркерами и fanIn паттерном
	u.asyncProcessor.ProcessURLsWithWorkers(codes, validator, processor)
}
