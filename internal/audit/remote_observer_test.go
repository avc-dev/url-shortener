package audit_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avc-dev/url-shortener/internal/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteObserver_Notify_SendsPostWithJSON(t *testing.T) {
	var received audit.Event
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &received))

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	obs := audit.NewRemoteObserver(srv.URL)
	event := audit.NewEvent(audit.ActionShorten, "user99", "https://example.com/long")

	err := obs.Notify(context.Background(), event)
	require.NoError(t, err)

	assert.Equal(t, event.Action, received.Action)
	assert.Equal(t, event.UserID, received.UserID)
	assert.Equal(t, event.URL, received.URL)
	assert.Equal(t, event.Ts, received.Ts)
}

func TestRemoteObserver_Notify_OmitsEmptyUserID(t *testing.T) {
	var raw map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &raw))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	obs := audit.NewRemoteObserver(srv.URL)
	require.NoError(t, obs.Notify(context.Background(), audit.NewEvent(audit.ActionFollow, "", "https://example.com")))

	_, hasUserID := raw["user_id"]
	assert.False(t, hasUserID, "user_id must be omitted when empty")
}

func TestRemoteObserver_Notify_InvalidURL_ReturnsError(t *testing.T) {
	obs := audit.NewRemoteObserver("://not-a-valid-url")
	err := obs.Notify(context.Background(), audit.NewEvent(audit.ActionShorten, "", "https://example.com"))
	assert.Error(t, err)
}

func TestRemoteObserver_Notify_ServerUnavailable_ReturnsError(t *testing.T) {
	// Стартуем сервер, запоминаем URL, закрываем — соединение будет отклонено
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	obs := audit.NewRemoteObserver(url)
	err := obs.Notify(context.Background(), audit.NewEvent(audit.ActionShorten, "", "https://example.com"))
	assert.Error(t, err)
}

func TestRemoteObserver_Notify_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Сервер вернул ошибку: RemoteObserver должен вернуть error
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	obs := audit.NewRemoteObserver(srv.URL)
	err := obs.Notify(context.Background(), audit.NewEvent(audit.ActionShorten, "", "https://example.com"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestRemoteObserver_Notify_ClientError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	obs := audit.NewRemoteObserver(srv.URL)
	err := obs.Notify(context.Background(), audit.NewEvent(audit.ActionShorten, "", "https://example.com"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "400")
}
