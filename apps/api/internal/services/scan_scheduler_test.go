package services

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
)

// MockSettingsRepoScheduler implements SettingsRepositoryInterface for scheduler tests
type MockSettingsRepoScheduler struct {
	mock.Mock
}

func (m *MockSettingsRepoScheduler) Set(ctx context.Context, setting *models.Setting) error {
	return nil
}
func (m *MockSettingsRepoScheduler) Get(ctx context.Context, key string) (*models.Setting, error) {
	return nil, nil
}
func (m *MockSettingsRepoScheduler) GetAll(ctx context.Context) ([]models.Setting, error) {
	return nil, nil
}
func (m *MockSettingsRepoScheduler) Delete(ctx context.Context, key string) error {
	return nil
}
func (m *MockSettingsRepoScheduler) GetString(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}
func (m *MockSettingsRepoScheduler) SetString(ctx context.Context, key, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}
func (m *MockSettingsRepoScheduler) GetInt(ctx context.Context, key string) (int, error) {
	return 0, nil
}
func (m *MockSettingsRepoScheduler) SetInt(ctx context.Context, key string, value int) error {
	return nil
}
func (m *MockSettingsRepoScheduler) GetBool(ctx context.Context, key string) (bool, error) {
	return false, nil
}
func (m *MockSettingsRepoScheduler) SetBool(ctx context.Context, key string, value bool) error {
	return nil
}

func setupScanScheduler(t *testing.T) (*ScanScheduler, *MockMovieRepoScanner, *MockSettingsRepoScheduler) {
	t.Helper()
	movieRepo := new(MockMovieRepoScanner)
	seriesRepo := new(MockSeriesRepoScanner)
	settingsRepo := new(MockSettingsRepoScheduler)
	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	scannerSvc := NewScannerService(movieRepo, seriesRepo, []string{}, hub, logger)

	// Default mock for FindAllWithFilePath
	movieRepo.On("FindAllWithFilePath", mock.Anything).Maybe().Return([]models.Movie{}, nil)

	scheduler := NewScanScheduler(scannerSvc, settingsRepo, logger)
	return scheduler, movieRepo, settingsRepo
}

func TestScanScheduler_Start_ManualNoTick(t *testing.T) {
	scheduler, _, settingsRepo := setupScanScheduler(t)

	// Settings returns "manual" — no auto-scan
	settingsRepo.On("GetString", mock.Anything, settingsKeyScanSchedule).Return("manual", nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		scheduler.Start(ctx)
		close(done)
	}()

	// Give scheduler time to start
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, ScheduleManual, scheduler.GetInterval())

	// Stop the scheduler
	cancel()
	select {
	case <-done:
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop in time")
	}
}

func TestScanScheduler_Reconfigure(t *testing.T) {
	scheduler, _, settingsRepo := setupScanScheduler(t)

	// Settings returns error (no setting) — defaults to manual
	settingsRepo.On("GetString", mock.Anything, settingsKeyScanSchedule).Return("", assert.AnError)
	settingsRepo.On("SetString", mock.Anything, settingsKeyScanSchedule, "hourly").Return(nil)
	settingsRepo.On("SetString", mock.Anything, settingsKeyScanSchedule, "manual").Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		scheduler.Start(ctx)
		close(done)
	}()

	// Wait for scheduler to start
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, ScheduleManual, scheduler.GetInterval())

	// Reconfigure to hourly
	err := scheduler.Reconfigure(ScheduleHourly)
	assert.NoError(t, err)
	assert.Equal(t, ScheduleHourly, scheduler.GetInterval())

	// Verify settings was persisted
	settingsRepo.AssertCalled(t, "SetString", mock.Anything, settingsKeyScanSchedule, "hourly")

	// Reconfigure back to manual
	err = scheduler.Reconfigure(ScheduleManual)
	assert.NoError(t, err)
	assert.Equal(t, ScheduleManual, scheduler.GetInterval())

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop")
	}
}

func TestScanScheduler_Stop(t *testing.T) {
	scheduler, _, settingsRepo := setupScanScheduler(t)

	settingsRepo.On("GetString", mock.Anything, settingsKeyScanSchedule).Return("manual", nil)

	ctx := context.Background()
	done := make(chan struct{})

	go func() {
		scheduler.Start(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	// Stop should be graceful
	scheduler.Stop()

	select {
	case <-done:
		// expected — scheduler exited
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop in time")
	}
}

func TestScanScheduler_SkipWhenScanActive(t *testing.T) {
	scheduler, _, settingsRepo := setupScanScheduler(t)

	settingsRepo.On("GetString", mock.Anything, settingsKeyScanSchedule).Return("manual", nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		scheduler.Start(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	// Simulate active scan
	scheduler.scannerService.mu.Lock()
	scheduler.scannerService.isScanning = true
	scheduler.scannerService.mu.Unlock()

	// Call onTick directly — should skip
	scheduler.onTick(ctx)

	// Clean up
	scheduler.scannerService.mu.Lock()
	scheduler.scannerService.isScanning = false
	scheduler.scannerService.mu.Unlock()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop")
	}
}

func TestScanScheduler_InvalidInterval(t *testing.T) {
	scheduler, _, settingsRepo := setupScanScheduler(t)

	settingsRepo.On("GetString", mock.Anything, settingsKeyScanSchedule).Return("manual", nil)

	err := scheduler.Reconfigure(ScanScheduleInterval("every_5_minutes"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SCANNER_SCHEDULE_INVALID")
}

func TestScanScheduler_GetTickerDuration(t *testing.T) {
	assert.Equal(t, 1*time.Hour, getTickerDuration(ScheduleHourly))
	assert.Equal(t, 24*time.Hour, getTickerDuration(ScheduleDaily))
	assert.Equal(t, time.Duration(0), getTickerDuration(ScheduleManual))
}

func TestScanScheduler_DoubleStop(t *testing.T) {
	scheduler, _, settingsRepo := setupScanScheduler(t)

	settingsRepo.On("GetString", mock.Anything, settingsKeyScanSchedule).Return("manual", nil)

	ctx := context.Background()
	done := make(chan struct{})

	go func() {
		scheduler.Start(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	// Double stop should not panic
	scheduler.Stop()
	scheduler.Stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop")
	}
}
