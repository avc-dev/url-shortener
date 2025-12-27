package service

import (
	"github.com/avc-dev/url-shortener/internal/model"
)

// URLValidator определяет функцию для валидации URL
type URLValidator func(code model.Code) bool

// AsyncURLProcessor обрабатывает URL асинхронно с использованием воркеров и fanIn паттерна
type AsyncURLProcessor struct{}

// NewAsyncURLProcessor создает новый AsyncURLProcessor
func NewAsyncURLProcessor() *AsyncURLProcessor {
	return &AsyncURLProcessor{}
}

// ProcessURLsWithWorkers обрабатывает список кодов с использованием воркеров и fanIn паттерна
// validator - функция для валидации каждого кода
// processor - функция для обработки валидных кодов
func (p *AsyncURLProcessor) ProcessURLsWithWorkers(
	codes []model.Code,
	validator URLValidator,
	processor func(validCodes []model.Code),
) {
	if len(codes) == 0 {
		return
	}

	// Количество воркеров для параллельной обработки
	numWorkers := 4
	if len(codes) < numWorkers {
		numWorkers = len(codes)
	}

	// Создаем канал для кодов
	codesChan := make(chan model.Code, len(codes))

	// Заполняем канал кодами
	go func() {
		defer close(codesChan)
		for _, code := range codes {
			codesChan <- code
		}
	}()

	// Создаем каналы для результатов от каждого воркера
	workerChannels := make([]chan model.Code, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workerChannels[i] = make(chan model.Code, len(codes)/numWorkers+1)
	}

	// Запускаем воркеров для валидации
	for i := 0; i < numWorkers; i++ {
		go func(workerID int, input <-chan model.Code, output chan<- model.Code) {
			defer close(output)
			for code := range input {
				// Выполняем валидацию
				if validator(code) {
					output <- code
				}
			}
		}(i, codesChan, workerChannels[i])
	}

	// FanIn: сливаем результаты от всех воркеров в один канал
	validCodesChan := make(chan model.Code, len(codes))
	go func() {
		defer close(validCodesChan)
		for _, workerChan := range workerChannels {
			for code := range workerChan {
				validCodesChan <- code
			}
		}
	}()

	// Собираем валидные коды
	var validCodes []model.Code
	for code := range validCodesChan {
		validCodes = append(validCodes, code)
	}

	// Выполняем обработку валидных кодов
	if len(validCodes) > 0 {
		processor(validCodes)
	}
}
