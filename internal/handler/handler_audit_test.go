package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/avc-dev/url-shortener/internal/audit"
	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// testAuditor — синхронный Auditor-stub для проверки событий в handler-тестах
type testAuditor struct {
	mu     sync.Mutex
	events []audit.Event
}

func (a *testAuditor) Notify(_ context.Context, event audit.Event) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, event)
}

func (a *testAuditor) snapshot() []audit.Event {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]audit.Event{}, a.events...)
}

// --- New() with auditor(s) ---

func TestNew_WithAuditor_StoresAuditor(t *testing.T) {
	mockUsecase := mocks.NewMockURLUsecase(t)
	aud := &testAuditor{}

	h := New(mockUsecase, zap.NewNop(), nil, aud)

	mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", "").
		Return("http://localhost/abc", nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("https://example.com"))
	h.CreateURL(httptest.NewRecorder(), req)

	require.Len(t, aud.snapshot(), 1)
}

func TestNew_WithMultipleAuditors_AllNotified(t *testing.T) {
	mockUsecase := mocks.NewMockURLUsecase(t)
	aud1 := &testAuditor{}
	aud2 := &testAuditor{}
	aud3 := &testAuditor{}

	h := New(mockUsecase, zap.NewNop(), nil, aud1, aud2, aud3)

	mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", "").
		Return("http://localhost/abc", nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("https://example.com"))
	h.CreateURL(httptest.NewRecorder(), req)

	require.Len(t, aud1.snapshot(), 1, "auditor 1 must be notified")
	require.Len(t, aud2.snapshot(), 1, "auditor 2 must be notified")
	require.Len(t, aud3.snapshot(), 1, "auditor 3 must be notified")

	// Все три получили одинаковое событие
	assert.Equal(t, aud1.snapshot()[0].URL, aud2.snapshot()[0].URL)
	assert.Equal(t, aud1.snapshot()[0].URL, aud3.snapshot()[0].URL)
}

// --- emitAudit nil safety ---

func TestEmitAudit_NilAuditor_NoPanic(t *testing.T) {
	mockUsecase := mocks.NewMockURLUsecase(t)
	h := New(mockUsecase, zap.NewNop(), nil) // без аудитора

	mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", "").
		Return("http://localhost/abc", nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("https://example.com"))
	assert.NotPanics(t, func() {
		h.CreateURL(httptest.NewRecorder(), req)
	})
}

// --- CreateURL emits shorten event ---

func TestCreateURL_EmitsAuditShortenEvent(t *testing.T) {
	mockUsecase := mocks.NewMockURLUsecase(t)
	aud := &testAuditor{}
	h := New(mockUsecase, zap.NewNop(), nil, aud)

	mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com/original", "").
		Return("http://localhost/abc", nil).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("https://example.com/original"))
	h.CreateURL(httptest.NewRecorder(), req)

	events := aud.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, audit.ActionShorten, events[0].Action)
	assert.Equal(t, "https://example.com/original", events[0].URL)
	assert.Empty(t, events[0].UserID)
	assert.Greater(t, events[0].TS, int64(0))
}

func TestCreateURL_Error_DoesNotEmitAuditEvent(t *testing.T) {
	mockUsecase := mocks.NewMockURLUsecase(t)
	aud := &testAuditor{}
	h := New(mockUsecase, zap.NewNop(), nil, aud)

	mockUsecase.EXPECT().
		CreateShortURLFromString("bad-url", "").
		Return("", usecase.ErrURLNotFound).
		Once()

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("bad-url"))
	h.CreateURL(httptest.NewRecorder(), req)

	assert.Empty(t, aud.snapshot(), "audit must not fire on handler error")
}

// --- CreateURLJSON emits shorten event ---

func TestCreateURLJSON_EmitsAuditShortenEvent(t *testing.T) {
	mockUsecase := mocks.NewMockURLUsecase(t)
	aud := &testAuditor{}
	h := New(mockUsecase, zap.NewNop(), nil, aud)

	originalURL := "https://example.com/json-original"
	mockUsecase.EXPECT().
		CreateShortURLFromString(originalURL, "").
		Return("http://localhost/xyz", nil).
		Once()

	body, err := json.Marshal(ShortenRequest{URL: originalURL})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.CreateURLJSON(httptest.NewRecorder(), req)

	events := aud.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, audit.ActionShorten, events[0].Action)
	assert.Equal(t, originalURL, events[0].URL)
	assert.Empty(t, events[0].UserID)
}

func TestCreateURLJSON_Error_DoesNotEmitAuditEvent(t *testing.T) {
	mockUsecase := mocks.NewMockURLUsecase(t)
	aud := &testAuditor{}
	h := New(mockUsecase, zap.NewNop(), nil, aud)

	mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", "").
		Return("", usecase.ErrURLNotFound).
		Once()

	body, _ := json.Marshal(ShortenRequest{URL: "https://example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
	h.CreateURLJSON(httptest.NewRecorder(), req)

	assert.Empty(t, aud.snapshot())
}

// --- GetURL emits follow event ---

func TestGetURL_EmitsAuditFollowEvent(t *testing.T) {
	mockUsecase := mocks.NewMockURLUsecase(t)
	aud := &testAuditor{}
	h := New(mockUsecase, zap.NewNop(), nil, aud)

	originalURL := "https://example.com/original-page"
	mockUsecase.EXPECT().
		GetOriginalURL("abc123").
		Return(originalURL, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.GetURL(httptest.NewRecorder(), req)

	events := aud.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, audit.ActionFollow, events[0].Action)
	assert.Equal(t, originalURL, events[0].URL)
	assert.Empty(t, events[0].UserID)
}

func TestGetURL_Error_DoesNotEmitAuditEvent(t *testing.T) {
	mockUsecase := mocks.NewMockURLUsecase(t)
	aud := &testAuditor{}
	h := New(mockUsecase, zap.NewNop(), nil, aud)

	mockUsecase.EXPECT().
		GetOriginalURL("notfound").
		Return("", usecase.ErrURLNotFound).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "notfound")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.GetURL(httptest.NewRecorder(), req)

	assert.Empty(t, aud.snapshot())
}
