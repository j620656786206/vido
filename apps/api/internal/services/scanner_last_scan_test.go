package services

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// memScanSettings is a map-backed settings store — only the string pair the
// last-scan record uses is real (bugfix-last-scan-never-shown).
type memScanSettings struct {
	mu      sync.Mutex
	values  map[string]string
	failSet bool
	getErrs int // the next N GetString calls fail transiently
}

func newMemScanSettings() *memScanSettings { return &memScanSettings{values: map[string]string{}} }

func (m *memScanSettings) Set(context.Context, *models.Setting) error           { return nil }
func (m *memScanSettings) Get(context.Context, string) (*models.Setting, error) { return nil, nil }
func (m *memScanSettings) GetAll(context.Context) ([]models.Setting, error)     { return nil, nil }
func (m *memScanSettings) Delete(context.Context, string) error                 { return nil }
func (m *memScanSettings) GetInt(context.Context, string) (int, error)          { return 0, nil }
func (m *memScanSettings) SetInt(context.Context, string, int) error            { return nil }
func (m *memScanSettings) GetBool(context.Context, string) (bool, error)        { return false, nil }
func (m *memScanSettings) SetBool(context.Context, string, bool) error          { return nil }
func (m *memScanSettings) GetString(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErrs > 0 {
		m.getErrs--
		return "", errors.New("database is locked")
	}
	v, ok := m.values[key]
	if !ok {
		return "", errors.New("setting with key " + key + " not found")
	}
	return v, nil
}
func (m *memScanSettings) SetString(_ context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failSet {
		return errors.New("disk full")
	}
	m.values[key] = value
	return nil
}

func TestScannerService_LastScan_NoneBeforeFirstScan(t *testing.T) {
	svc, _, _ := setupScannerService(t, []string{t.TempDir()})
	svc.SetSettingsRepo(newMemScanSettings())
	assert.Nil(t, svc.GetLastScan(context.Background()))
}

func TestScannerService_LastScan_RecordedAndPersisted(t *testing.T) {
	dir := t.TempDir()
	createVideoFiles(t, dir, []string{"a.mkv", "b.mp4"})
	svc, movieRepo, _ := setupScannerService(t, []string{dir})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)
	settings := newMemScanSettings()
	svc.SetSettingsRepo(settings)

	before := time.Now()
	_, err := svc.StartScan(context.Background())
	require.NoError(t, err)

	last := svc.GetLastScan(context.Background())
	require.NotNil(t, last)
	assert.Equal(t, 2, last.FilesFound)
	assert.False(t, last.CompletedAt.Before(before), "completed_at is this scan's end")
	assert.GreaterOrEqual(t, last.DurationMs, int64(0))

	// A restarted server (fresh service, same settings) still knows it.
	fresh, _, _ := setupScannerService(t, []string{dir})
	fresh.SetSettingsRepo(settings)
	reloaded := fresh.GetLastScan(context.Background())
	require.NotNil(t, reloaded)
	assert.Equal(t, 2, reloaded.FilesFound)
	assert.True(t, reloaded.CompletedAt.Equal(last.CompletedAt))

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(settings.values[settingsKeyScanLastResult]), &raw))
	assert.ElementsMatch(t, []string{"completed_at", "files_found", "duration_ms"}, keysOf(raw))
}

func TestScannerService_LastScan_CancelledScanKeepsPreviousRecord(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		require.NoError(t, os.WriteFile(filepath.Join(dir, string(rune('a'+i))+".mkv"), []byte("x"), 0644))
	}
	svc, movieRepo, _ := setupScannerService(t, []string{dir})
	settings := newMemScanSettings()
	prev := LastScanSummary{CompletedAt: time.Date(2026, 3, 22, 14, 30, 0, 0, time.UTC), FilesFound: 1247, DurationMs: 192000}
	b, _ := json.Marshal(prev)
	settings.values[settingsKeyScanLastResult] = string(b)
	svc.SetSettingsRepo(settings)

	var once sync.Once
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).
		Run(func(mock.Arguments) { once.Do(func() { require.NoError(t, svc.CancelScan()) }) }).
		Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	_, err := svc.StartScan(context.Background())
	require.NoError(t, err)

	last := svc.GetLastScan(context.Background())
	require.NotNil(t, last)
	assert.Equal(t, 1247, last.FilesFound, "a cancelled scan is not a finished scan")
}

func TestScannerService_LastScan_PersistFailureStillRemembered(t *testing.T) {
	dir := t.TempDir()
	createVideoFiles(t, dir, []string{"a.mkv"})
	svc, movieRepo, _ := setupScannerService(t, []string{dir})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)
	settings := newMemScanSettings()
	settings.failSet = true
	svc.SetSettingsRepo(settings)

	_, err := svc.StartScan(context.Background())
	require.NoError(t, err, "a failed save must not fail the scan")
	last := svc.GetLastScan(context.Background())
	require.NotNil(t, last)
	assert.Equal(t, 1, last.FilesFound)
}

func seedLastScan(t *testing.T, settings *memScanSettings, files int) {
	t.Helper()
	b, err := json.Marshal(LastScanSummary{CompletedAt: time.Date(2026, 3, 22, 14, 30, 0, 0, time.UTC), FilesFound: files, DurationMs: 192000})
	require.NoError(t, err)
	settings.values[settingsKeyScanLastResult] = string(b)
}

// Review M1: a transient read error (busy DB, a request cancelled mid-read)
// must not be remembered as "never scanned" until the next restart.
func TestScannerService_LastScan_TransientReadErrorIsRetried(t *testing.T) {
	svc, _, _ := setupScannerService(t, []string{t.TempDir()})
	settings := newMemScanSettings()
	seedLastScan(t, settings, 1247)
	settings.getErrs = 1
	svc.SetSettingsRepo(settings)

	assert.Nil(t, svc.GetLastScan(context.Background()), "the failed read shows nothing this time")
	again := svc.GetLastScan(context.Background())
	require.NotNil(t, again, "the next read tries the store again")
	assert.Equal(t, 1247, again.FilesFound)
}

func TestScannerService_LastScan_CancelledRequestContextStillReads(t *testing.T) {
	svc, _, _ := setupScannerService(t, []string{t.TempDir()})
	settings := newMemScanSettings()
	seedLastScan(t, settings, 7)
	svc.SetSettingsRepo(settings)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NotNil(t, svc.GetLastScan(ctx))
}

func TestScannerService_LastScan_EmptyRecordIsNoRecord(t *testing.T) {
	svc, _, _ := setupScannerService(t, []string{t.TempDir()})
	settings := newMemScanSettings()
	settings.values[settingsKeyScanLastResult] = "{}"
	svc.SetSettingsRepo(settings)
	assert.Nil(t, svc.GetLastScan(context.Background()), "a zero completed_at would read 0001-01-01")
}

// Review M2: every folder unreachable (NAS mount dropped) is not a finished
// scan — the previous record stays instead of 「0 檔案」.
func TestScannerService_LastScan_AllFoldersMissingKeepsPreviousRecord(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone")
	svc, _, _ := setupScannerService(t, []string{missing})
	settings := newMemScanSettings()
	seedLastScan(t, settings, 1247)
	svc.SetSettingsRepo(settings)

	_, err := svc.StartScan(context.Background())
	require.NoError(t, err)
	last := svc.GetLastScan(context.Background())
	require.NotNil(t, last)
	assert.Equal(t, 1247, last.FilesFound)
}

func keysOf(m map[string]interface{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
