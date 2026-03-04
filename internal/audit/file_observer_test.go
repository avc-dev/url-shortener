package audit_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/avc-dev/url-shortener/internal/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileObserver_Notify_CreatesFileAndWritesJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	obs := audit.NewFileObserver(path)

	event := audit.NewEvent(audit.ActionShorten, "user1", "https://example.com/path")
	err := obs.Notify(context.Background(), event)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var got audit.Event
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, event.Action, got.Action)
	assert.Equal(t, event.UserID, got.UserID)
	assert.Equal(t, event.URL, got.URL)
	assert.Equal(t, event.TS, got.TS)
}

func TestFileObserver_Notify_AppendsNewLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	obs := audit.NewFileObserver(path)

	events := []audit.Event{
		audit.NewEvent(audit.ActionShorten, "u1", "https://example.com/1"),
		audit.NewEvent(audit.ActionFollow, "u2", "https://example.com/2"),
		audit.NewEvent(audit.ActionShorten, "", "https://example.com/3"),
	}
	for _, e := range events {
		require.NoError(t, obs.Notify(context.Background(), e))
	}

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	var lines []audit.Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e audit.Event
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &e))
		lines = append(lines, e)
	}
	require.NoError(t, scanner.Err())

	require.Len(t, lines, 3)
	assert.Equal(t, "https://example.com/1", lines[0].URL)
	assert.Equal(t, "https://example.com/2", lines[1].URL)
	assert.Equal(t, "https://example.com/3", lines[2].URL)
}

func TestFileObserver_Notify_OmitsEmptyUserID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	obs := audit.NewFileObserver(path)

	require.NoError(t, obs.Notify(context.Background(), audit.NewEvent(audit.ActionFollow, "", "https://example.com")))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	// user_id должен отсутствовать в JSON при пустом значении (omitempty)
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	_, hasUserID := raw["user_id"]
	assert.False(t, hasUserID, "user_id should be omitted when empty")
}

func TestFileObserver_Notify_InvalidPath_ReturnsError(t *testing.T) {
	// Несуществующий промежуточный каталог — OpenFile должен вернуть ошибку
	path := filepath.Join(t.TempDir(), "nonexistent_subdir", "audit.log")
	obs := audit.NewFileObserver(path)

	err := obs.Notify(context.Background(), audit.NewEvent(audit.ActionShorten, "", "https://example.com"))
	assert.Error(t, err)
}

func TestFileObserver_Notify_ConcurrentWritesSafe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	obs := audit.NewFileObserver(path)

	const workers = 20
	errs := make([]error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			// Каждая горутина пишет в свой индекс — гонки нет.
			errs[i] = obs.Notify(context.Background(), audit.NewEvent(audit.ActionShorten, "", "https://example.com"))
		}()
	}
	wg.Wait()
	// Assertions — только в основной горутине, после WaitGroup.
	for _, err := range errs {
		assert.NoError(t, err)
	}

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	var count int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var e audit.Event
		assert.NoError(t, json.Unmarshal(line, &e), "each line must be valid JSON")
		count++
	}
	require.NoError(t, scanner.Err())
	assert.Equal(t, workers, count, "every concurrent write must produce exactly one line")
}
