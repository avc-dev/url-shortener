package audit_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/avc-dev/url-shortener/internal/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// --- helpers ---

// recordingObserver потокобезопасно накапливает полученные события.
type recordingObserver struct {
	mu     sync.Mutex
	events []audit.Event
}

func (r *recordingObserver) Notify(_ context.Context, event audit.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return nil
}

func (r *recordingObserver) snapshot() []audit.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]audit.Event{}, r.events...)
}

// funcObserver позволяет задать произвольную логику Observer через замыкание.
type funcObserver struct {
	fn func(context.Context, audit.Event) error
}

func (f *funcObserver) Notify(ctx context.Context, event audit.Event) error {
	return f.fn(ctx, event)
}

// --- NewEvent ---

func TestNewEvent_FieldsAndTimestamp(t *testing.T) {
	event := audit.NewEvent(audit.ActionShorten, "user42", "https://example.com/long")
	assert.Equal(t, audit.ActionShorten, event.Action)
	assert.Equal(t, "user42", event.UserID)
	assert.Equal(t, "https://example.com/long", event.URL)
	assert.Greater(t, event.Ts, int64(0))
}

func TestNewEvent_EmptyUserID(t *testing.T) {
	event := audit.NewEvent(audit.ActionFollow, "", "https://example.com")
	assert.Equal(t, audit.ActionFollow, event.Action)
	assert.Empty(t, event.UserID)
}

func TestActionConstants(t *testing.T) {
	assert.Equal(t, "shorten", audit.ActionShorten)
	assert.Equal(t, "follow", audit.ActionFollow)
}

// --- Subject ---

func TestSubject_Notify_CallsObserver(t *testing.T) {
	subject := audit.NewSubject(zap.NewNop())
	obs := &recordingObserver{}
	subject.Register(obs)

	event := audit.NewEvent(audit.ActionShorten, "u1", "https://example.com")
	subject.Notify(context.Background(), event)
	subject.Close() // детерминированно ждём завершения горутины

	events := obs.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, audit.ActionShorten, events[0].Action)
	assert.Equal(t, "u1", events[0].UserID)
	assert.Equal(t, "https://example.com", events[0].URL)
}

func TestSubject_Notify_CallsAllObservers(t *testing.T) {
	subject := audit.NewSubject(zap.NewNop())
	obs1 := &recordingObserver{}
	obs2 := &recordingObserver{}
	subject.Register(obs1)
	subject.Register(obs2)

	subject.Notify(context.Background(), audit.NewEvent(audit.ActionFollow, "", "https://example.com"))
	subject.Close()

	require.Len(t, obs1.snapshot(), 1)
	require.Len(t, obs2.snapshot(), 1)
}

func TestSubject_Notify_NoObservers_NoPanic(t *testing.T) {
	subject := audit.NewSubject(zap.NewNop())
	assert.NotPanics(t, func() {
		subject.Notify(context.Background(), audit.NewEvent(audit.ActionShorten, "", "https://example.com"))
		subject.Close()
	})
}

func TestSubject_Notify_ObserverError_IsLogged(t *testing.T) {
	core, logs := observer.New(zapcore.ErrorLevel)
	logger := zap.New(core)
	subject := audit.NewSubject(logger)

	subject.Register(&funcObserver{fn: func(_ context.Context, _ audit.Event) error {
		return errors.New("sink unavailable")
	}})

	subject.Notify(context.Background(), audit.NewEvent(audit.ActionShorten, "", "https://example.com"))
	// Close() возвращается только после того, как горутина полностью завершилась,
	// включая вызов logger.Error — никакого time.Sleep не нужно.
	subject.Close()

	require.Equal(t, 1, logs.Len(), "expected one error log entry")
	assert.Equal(t, "audit observer failed", logs.All()[0].Message)
}

func TestSubject_Notify_ErrorInOneObserver_OtherStillCalled(t *testing.T) {
	subject := audit.NewSubject(zap.NewNop())
	subject.Register(&funcObserver{fn: func(_ context.Context, _ audit.Event) error {
		return errors.New("error")
	}})
	goodObs := &recordingObserver{}
	subject.Register(goodObs)

	subject.Notify(context.Background(), audit.NewEvent(audit.ActionFollow, "", "https://example.com"))
	subject.Close()

	require.Len(t, goodObs.snapshot(), 1)
}

func TestSubject_Notify_MultipleEvents(t *testing.T) {
	subject := audit.NewSubject(zap.NewNop())
	obs := &recordingObserver{}
	subject.Register(obs)

	const n = 5
	for i := 0; i < n; i++ {
		subject.Notify(context.Background(), audit.NewEvent(audit.ActionShorten, "", "https://example.com"))
	}
	subject.Close()
	assert.Len(t, obs.snapshot(), n)
}

func TestSubject_Close_IsIdempotent(t *testing.T) {
	subject := audit.NewSubject(zap.NewNop())
	subject.Register(&recordingObserver{})
	subject.Notify(context.Background(), audit.NewEvent(audit.ActionShorten, "", "https://example.com"))
	subject.Close()
	// повторный Close не должен паниковать
	assert.NotPanics(t, subject.Close)
}
