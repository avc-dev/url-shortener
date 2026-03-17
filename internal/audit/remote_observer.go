package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RemoteObserver отправляет события аудита на удалённый сервер методом POST.
type RemoteObserver struct {
	client *http.Client
	url    string
}

// NewRemoteObserver создаёт RemoteObserver с таймаутом 5 секунд.
func NewRemoteObserver(url string) *RemoteObserver {
	return &RemoteObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Notify сериализует событие в JSON и отправляет его POST-запросом.
// Возвращает ошибку при проблемах сети или HTTP-статусе >= 400.
func (r *RemoteObserver) Notify(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("send audit event: %w", err)
	}
	defer resp.Body.Close()
	// Вычитываем тело ответа, чтобы TCP-соединение вернулось в пул клиента.
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("audit server returned status %d", resp.StatusCode)
	}
	return nil
}
