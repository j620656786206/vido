package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

const (
	settingsKeyBackupSchedule = "backup_schedule"
)

// BackupSchedule represents the backup schedule configuration
type BackupSchedule struct {
	Enabled   bool   `json:"enabled"`
	Frequency string `json:"frequency"` // "daily", "weekly", "disabled"
	Hour      int    `json:"hour"`      // 0-23
	DayOfWeek int    `json:"dayOfWeek"` // 0=Sunday, 1=Monday, ... (only for weekly)
}

// BackupScheduleResponse extends BackupSchedule with computed fields
type BackupScheduleResponse struct {
	BackupSchedule
	NextBackupAt *time.Time `json:"nextBackupAt,omitempty"`
	LastBackupAt *time.Time `json:"lastBackupAt,omitempty"`
}

// RetentionResult contains the outcome of a retention policy application
type RetentionResult struct {
	Deleted int `json:"deleted"`
	Kept    int `json:"kept"`
}

// BackupSchedulerInterface defines the contract for backup scheduling operations
type BackupSchedulerInterface interface {
	Start(ctx context.Context)
	Stop()
	SetSchedule(ctx context.Context, schedule BackupSchedule) error
	GetSchedule(ctx context.Context) (*BackupScheduleResponse, error)
}

// BackupScheduler manages automatic backup scheduling
type BackupScheduler struct {
	backupService BackupServiceInterface
	settingsRepo  repository.SettingsRepositoryInterface
	backupRepo    repository.BackupRepositoryInterface
	mu            sync.Mutex
	stopCh        chan struct{}
	stopped       bool
	lastRunDate   string // "2006-01-02" format to prevent duplicate runs
}

// Compile-time interface verification
var _ BackupSchedulerInterface = (*BackupScheduler)(nil)

// NewBackupScheduler creates a new BackupScheduler
func NewBackupScheduler(
	backupService BackupServiceInterface,
	settingsRepo repository.SettingsRepositoryInterface,
	backupRepo repository.BackupRepositoryInterface,
) *BackupScheduler {
	return &BackupScheduler{
		backupService: backupService,
		settingsRepo:  settingsRepo,
		backupRepo:    backupRepo,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the scheduler loop that checks schedule every minute
func (s *BackupScheduler) Start(ctx context.Context) {
	slog.Info("Backup scheduler started")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Backup scheduler stopped (context cancelled)")
			return
		case <-s.stopCh:
			slog.Info("Backup scheduler stopped (stop signal)")
			return
		case now := <-ticker.C:
			s.checkAndRun(ctx, now)
		}
	}
}

// Stop gracefully stops the scheduler
func (s *BackupScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.stopped {
		s.stopped = true
		close(s.stopCh)
	}
}

// SetSchedule saves the backup schedule configuration
func (s *BackupScheduler) SetSchedule(ctx context.Context, schedule BackupSchedule) error {
	if err := validateSchedule(schedule); err != nil {
		return err
	}

	data, err := json.Marshal(schedule)
	if err != nil {
		return fmt.Errorf("marshal schedule: %w", err)
	}

	if err := s.settingsRepo.SetString(ctx, settingsKeyBackupSchedule, string(data)); err != nil {
		return fmt.Errorf("save schedule: %w", err)
	}

	slog.Info("Backup schedule updated", "enabled", schedule.Enabled, "frequency", schedule.Frequency, "hour", schedule.Hour)
	return nil
}

// GetSchedule retrieves the current backup schedule configuration
func (s *BackupScheduler) GetSchedule(ctx context.Context) (*BackupScheduleResponse, error) {
	schedule, err := s.loadSchedule(ctx)
	if err != nil {
		return nil, err
	}

	resp := &BackupScheduleResponse{
		BackupSchedule: *schedule,
	}

	if schedule.Enabled {
		next := s.calculateNextBackup(time.Now(), schedule)
		resp.NextBackupAt = &next
	}

	// Get last backup time
	backups, err := s.backupRepo.List(ctx)
	if err == nil && len(backups) > 0 {
		resp.LastBackupAt = &backups[0].CreatedAt
	}

	return resp, nil
}

func (s *BackupScheduler) loadSchedule(ctx context.Context) (*BackupSchedule, error) {
	data, err := s.settingsRepo.GetString(ctx, settingsKeyBackupSchedule)
	if err != nil {
		// No schedule configured yet — return default (disabled)
		return &BackupSchedule{
			Enabled:   false,
			Frequency: "disabled",
			Hour:      3,
			DayOfWeek: 0,
		}, nil
	}

	var schedule BackupSchedule
	if err := json.Unmarshal([]byte(data), &schedule); err != nil {
		return nil, fmt.Errorf("parse schedule: %w", err)
	}
	return &schedule, nil
}

func validateSchedule(s BackupSchedule) error {
	if s.Frequency != "daily" && s.Frequency != "weekly" && s.Frequency != "disabled" {
		return fmt.Errorf("SCHEDULE_INVALID: frequency must be daily, weekly, or disabled")
	}
	if s.Hour < 0 || s.Hour > 23 {
		return fmt.Errorf("SCHEDULE_INVALID: hour must be 0-23")
	}
	if s.Frequency == "weekly" && (s.DayOfWeek < 0 || s.DayOfWeek > 6) {
		return fmt.Errorf("SCHEDULE_INVALID: dayOfWeek must be 0-6 (Sunday-Saturday)")
	}
	return nil
}

func (s *BackupScheduler) checkAndRun(ctx context.Context, now time.Time) {
	schedule, err := s.loadSchedule(ctx)
	if err != nil {
		slog.Error("Failed to load backup schedule", "error", err)
		return
	}
	if !schedule.Enabled {
		return
	}

	if !s.shouldRunNow(now, schedule) {
		return
	}

	slog.Info("Starting scheduled backup", "frequency", schedule.Frequency, "hour", schedule.Hour)
	backup, err := s.backupService.CreateBackup(ctx)
	if err != nil {
		slog.Error("Scheduled backup failed", "error", err)
		return
	}

	slog.Info("Scheduled backup completed", "id", backup.ID, "filename", backup.Filename)

	// Apply retention policy after successful backup
	result, err := s.ApplyRetentionPolicy(ctx)
	if err != nil {
		slog.Error("Retention policy failed", "error", err)
	} else if result.Deleted > 0 {
		slog.Info("Retention policy applied", "deleted", result.Deleted, "kept", result.Kept)
	}
}

func (s *BackupScheduler) shouldRunNow(now time.Time, schedule *BackupSchedule) bool {
	// Only run at the configured hour (minute 0)
	if now.Hour() != schedule.Hour || now.Minute() != 0 {
		return false
	}

	// For weekly: check day of week
	if schedule.Frequency == "weekly" && int(now.Weekday()) != schedule.DayOfWeek {
		return false
	}

	// Prevent duplicate runs within the same day
	dateStr := now.Format("2006-01-02")
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastRunDate == dateStr {
		return false
	}
	s.lastRunDate = dateStr
	return true
}

func (s *BackupScheduler) calculateNextBackup(now time.Time, schedule *BackupSchedule) time.Time {
	// Start with today at the configured hour
	next := time.Date(now.Year(), now.Month(), now.Day(), schedule.Hour, 0, 0, 0, now.Location())

	switch schedule.Frequency {
	case "daily":
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
	case "weekly":
		// Find next occurrence of the configured day
		daysUntil := (schedule.DayOfWeek - int(next.Weekday()) + 7) % 7
		if daysUntil == 0 && !next.After(now) {
			daysUntil = 7
		}
		next = next.AddDate(0, 0, daysUntil)
	}

	return next
}

// ApplyRetentionPolicy cleans up old backups according to retention rules
func (s *BackupScheduler) ApplyRetentionPolicy(ctx context.Context) (*RetentionResult, error) {
	backups, err := s.backupRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}

	result := &RetentionResult{}

	// Separate auto-snapshots (never delete) from regular backups
	var regular []struct {
		id       string
		filename string
		date     time.Time
	}

	for _, b := range backups {
		// Never delete auto-snapshots or non-completed backups
		if isAutoSnapshot(b.Filename) || b.Status != models.BackupStatusCompleted {
			result.Kept++
			continue
		}
		regular = append(regular, struct {
			id       string
			filename string
			date     time.Time
		}{b.ID, b.Filename, b.CreatedAt})
	}

	// Keep first 7 (most recent — list is sorted by created_at DESC)
	const dailyRetention = 7
	const weeklyRetention = 4

	keep := make(map[string]bool)

	for i, b := range regular {
		if i < dailyRetention {
			keep[b.id] = true
		}
	}

	// For remaining, keep 1 per week for the last 4 weeks
	if len(regular) > dailyRetention {
		weeksSeen := 0
		lastWeek := ""
		for _, b := range regular[dailyRetention:] {
			if weeksSeen >= weeklyRetention {
				break
			}
			year, week := b.date.ISOWeek()
			weekKey := fmt.Sprintf("%d-W%02d", year, week)
			if weekKey != lastWeek {
				keep[b.id] = true
				lastWeek = weekKey
				weeksSeen++
			}
		}
	}

	// Delete backups not in keep set
	for _, b := range regular {
		if keep[b.id] {
			result.Kept++
			continue
		}
		if err := s.backupService.DeleteBackup(ctx, b.id); err != nil {
			slog.Error("Failed to delete expired backup", "id", b.id, "error", err)
			continue
		}
		result.Deleted++
	}

	return result, nil
}

func isAutoSnapshot(filename string) bool {
	return strings.HasPrefix(filename, "vido-auto-snapshot-before-restore-")
}
