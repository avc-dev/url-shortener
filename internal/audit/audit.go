// Package audit реализует систему аудита запросов по паттерну «Наблюдатель».
// Subject рассылает события всем зарегистрированным Observer асинхронно;
// ошибки наблюдателей логируются, но не влияют на обработку HTTP-запросов.
package audit

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	ActionShorten = "shorten"
	ActionFollow  = "follow"
)

// Event представляет событие аудита.
type Event struct {
	Action    string `json:"action"`
	UserID    string `json:"user_id,omitempty"`
	URL       string `json:"url"`
	ShortCode string `json:"short_code,omitempty"`
	TS        int64  `json:"ts"`
}

// NewEvent создаёт событие аудита с текущим unix-временем.
func NewEvent(action, userID, url string) Event {
	return Event{
		TS:     time.Now().Unix(),
		Action: action,
		UserID: userID,
		URL:    url,
	}
}

// NewFollowEvent создаёт событие аудита для операции перехода по короткому коду.
// Сохраняет как короткий код (для трассировки), так и оригинальный URL (цель перехода).
func NewFollowEvent(userID, shortCode, originalURL string) Event {
	return Event{
		TS:        time.Now().Unix(),
		Action:    ActionFollow,
		UserID:    userID,
		ShortCode: shortCode,
		URL:       originalURL,
	}
}

// Observer — интерфейс приёмника событий аудита (низкоуровневый, возвращает error).
type Observer interface {
	Notify(ctx context.Context, event Event) error
}

// Notifier — высокоуровневый интерфейс аудита для потребителей (handler, grpchandler).
// В отличие от Observer не возвращает ошибку: Subject поглощает её внутри.
// Определён здесь как единственный источник истины, чтобы handler и grpchandler
// не дублировали одинаковое объявление интерфейса.
type Notifier interface {
	Notify(ctx context.Context, event Event)
}

// Subject хранит список Observer и асинхронно рассылает им события.
// Безопасен для конкурентного использования.
type Subject struct {
	logger    *zap.Logger
	observers []Observer
	wg        sync.WaitGroup
	mu        sync.RWMutex
}

// NewSubject создаёт новый Subject.
func NewSubject(logger *zap.Logger) *Subject {
	return &Subject{logger: logger}
}

// Register добавляет наблюдателя. Может вызываться конкурентно.
func (s *Subject) Register(o Observer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observers = append(s.observers, o)
}

// Notify асинхронно уведомляет всех зарегистрированных наблюдателей о событии.
//
// Намеренно использует context.Background() для каждой горутины: аудит должен
// быть записан независимо от отмены HTTP/gRPC запроса.
// Ошибки наблюдателей логируются и не прерывают остальных.
func (s *Subject) Notify(_ context.Context, event Event) {
	s.mu.RLock()
	observers := s.observers // snapshot среза под RLock
	s.mu.RUnlock()

	for _, o := range observers {
		observer := o
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := observer.Notify(context.Background(), event); err != nil {
				s.logger.Error("audit observer failed", zap.Error(err))
			}
		}()
	}
}

// Close ожидает завершения всех запущенных горутин наблюдателей.
// Вызывайте при остановке приложения, чтобы гарантировать доставку
// всех событий аудита до выхода процесса.
func (s *Subject) Close() {
	s.wg.Wait()
}
