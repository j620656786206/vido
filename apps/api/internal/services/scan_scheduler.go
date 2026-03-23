package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vido/api/internal/repository"
)

const (
	settingsKeyScanSchedule = "scan_schedule"
)

// ScanScheduleInterval represents the scan schedule interval
type ScanScheduleInterval string

const (
	ScheduleManual ScanScheduleInterval = "manual"
	ScheduleHourly ScanScheduleInterval = "hourly"
	ScheduleDaily  ScanScheduleInterval = "daily"
)

// ValidScanScheduleIntervals contains all valid interval values
var ValidScanScheduleIntervals = map[ScanScheduleInterval]bool{
	ScheduleManual: true,
	ScheduleHourly: true,
	ScheduleDaily:  true,
}

// ScanSchedulerInterface defines the contract for scan scheduling operations
type ScanSchedulerInterface interface {
	Start(ctx context.Context)
	Stop()
	Reconfigure(interval ScanScheduleInterval) error
	GetInterval() ScanScheduleInterval
}

// ScanScheduler manages automatic scan scheduling using time.Ticker
type ScanScheduler struct {
	scannerService *ScannerService
	settingsRepo   repository.SettingsRepositoryInterface
	logger         *slog.Logger

	mu       sync.Mutex
	ticker   *time.Ticker
	done     chan struct{}
	interval ScanScheduleInterval
	running  bool
}

// Compile-time interface verification
var _ ScanSchedulerInterface = (*ScanScheduler)(nil)

// NewScanScheduler creates a new ScanScheduler
func NewScanScheduler(
	scannerService *ScannerService,
	settingsRepo repository.SettingsRepositoryInterface,
	logger *slog.Logger,
) *ScanScheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ScanScheduler{
		scannerService: scannerService,
		settingsRepo:   settingsRepo,
		logger:         logger,
		interval:       ScheduleManual,
	}
}

// Start loads the schedule from settings and begins the ticker goroutine.
// This method blocks until ctx is cancelled or Stop() is called.
func (s *ScanScheduler) Start(ctx context.Context) {
	// Load schedule from settings
	interval := s.loadScheduleFromSettings(ctx)
	s.mu.Lock()
	s.interval = interval
	s.done = make(chan struct{})
	s.running = true
	s.mu.Unlock()

	s.logger.Info("Scan scheduler started", "interval", interval)

	if interval == ScheduleManual {
		// No ticker needed for manual mode — just wait for stop/ctx
		select {
		case <-ctx.Done():
			s.logger.Info("Scan scheduler stopped (context cancelled)")
		case <-s.done:
			s.logger.Info("Scan scheduler stopped (stop signal)")
		}
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return
	}

	duration := getTickerDuration(interval)
	s.mu.Lock()
	s.ticker = time.NewTicker(duration)
	s.mu.Unlock()

	s.runTickerLoop(ctx)
}

// Stop gracefully stops the scheduler
func (s *ScanScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
	if s.done != nil {
		select {
		case <-s.done:
			// already closed
		default:
			close(s.done)
		}
	}
}

// Reconfigure validates the interval, persists it to settings, stops the old ticker, and starts a new one
func (s *ScanScheduler) Reconfigure(interval ScanScheduleInterval) error {
	if !ValidScanScheduleIntervals[interval] {
		return fmt.Errorf("SCANNER_SCHEDULE_INVALID: unrecognized schedule interval: %s", interval)
	}

	// Persist to settings
	ctx := context.Background()
	if err := s.settingsRepo.SetString(ctx, settingsKeyScanSchedule, string(interval)); err != nil {
		return fmt.Errorf("failed to persist scan schedule: %w", err)
	}

	s.mu.Lock()
	oldInterval := s.interval
	s.interval = interval

	// Stop old ticker if any
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}

	// Start new ticker if not manual
	if interval != ScheduleManual {
		duration := getTickerDuration(interval)
		s.ticker = time.NewTicker(duration)
	}
	s.mu.Unlock()

	s.logger.Info("Scan schedule reconfigured", "old_interval", oldInterval, "new_interval", interval)
	return nil
}

// GetInterval returns the current schedule interval
func (s *ScanScheduler) GetInterval() ScanScheduleInterval {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.interval
}

// loadScheduleFromSettings loads the schedule preference from the settings repository
func (s *ScanScheduler) loadScheduleFromSettings(ctx context.Context) ScanScheduleInterval {
	value, err := s.settingsRepo.GetString(ctx, settingsKeyScanSchedule)
	if err != nil {
		s.logger.Info("No scan schedule configured, defaulting to manual")
		return ScheduleManual
	}

	interval := ScanScheduleInterval(value)
	if !ValidScanScheduleIntervals[interval] {
		s.logger.Warn("Invalid scan schedule in settings, defaulting to manual", "value", value)
		return ScheduleManual
	}

	return interval
}

// runTickerLoop runs the ticker loop, checking for scan triggers
func (s *ScanScheduler) runTickerLoop(ctx context.Context) {
	defer func() {
		s.mu.Lock()
		s.running = false
		if s.ticker != nil {
			s.ticker.Stop()
			s.ticker = nil
		}
		s.mu.Unlock()
	}()

	for {
		s.mu.Lock()
		ticker := s.ticker
		done := s.done
		s.mu.Unlock()

		if ticker == nil {
			// Manual mode or stopped — wait for done/ctx
			select {
			case <-ctx.Done():
				s.logger.Info("Scan scheduler stopped (context cancelled)")
				return
			case <-done:
				s.logger.Info("Scan scheduler stopped (stop signal)")
				return
			}
		}

		select {
		case <-ctx.Done():
			s.logger.Info("Scan scheduler stopped (context cancelled)")
			return
		case <-done:
			s.logger.Info("Scan scheduler stopped (stop signal)")
			return
		case <-ticker.C:
			s.onTick(ctx)
		}
	}
}

// onTick handles a scheduled scan trigger
func (s *ScanScheduler) onTick(ctx context.Context) {
	// Task 3: Check if a scan is already active (mutex conflict handling)
	if s.scannerService.IsScanActive() {
		s.logger.Info("Scheduled scan skipped — manual scan in progress")
		return
	}

	s.logger.Info("Starting scheduled scan")
	go func() {
		result, err := s.scannerService.StartScan(context.Background())
		if err != nil {
			s.logger.Error("Scheduled scan failed", "error", err)
			return
		}
		if result != nil {
			s.logger.Info("Scheduled scan completed",
				"files_found", result.FilesFound,
				"files_created", result.FilesCreated,
				"files_updated", result.FilesUpdated,
				"duration", result.Duration,
			)
		}
	}()
}

// getTickerDuration converts a schedule interval to a time.Duration
func getTickerDuration(interval ScanScheduleInterval) time.Duration {
	switch interval {
	case ScheduleHourly:
		return 1 * time.Hour
	case ScheduleDaily:
		return 24 * time.Hour
	default:
		return 0
	}
}
