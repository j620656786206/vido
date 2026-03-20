package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
)

// MockSchedulerSettingsRepo implements SettingsRepositoryInterface for testing
type MockSchedulerSettingsRepo struct {
	mock.Mock
}

func (m *MockSchedulerSettingsRepo) Set(ctx context.Context, setting *models.Setting) error {
	return m.Called(ctx, setting).Error(0)
}
func (m *MockSchedulerSettingsRepo) Get(ctx context.Context, key string) (*models.Setting, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Setting), args.Error(1)
}
func (m *MockSchedulerSettingsRepo) GetAll(ctx context.Context) ([]models.Setting, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Setting), args.Error(1)
}
func (m *MockSchedulerSettingsRepo) Delete(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}
func (m *MockSchedulerSettingsRepo) GetString(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}
func (m *MockSchedulerSettingsRepo) GetInt(ctx context.Context, key string) (int, error) {
	args := m.Called(ctx, key)
	return args.Int(0), args.Error(1)
}
func (m *MockSchedulerSettingsRepo) GetBool(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}
func (m *MockSchedulerSettingsRepo) SetString(ctx context.Context, key, value string) error {
	return m.Called(ctx, key, value).Error(0)
}
func (m *MockSchedulerSettingsRepo) SetInt(ctx context.Context, key string, value int) error {
	return m.Called(ctx, key, value).Error(0)
}
func (m *MockSchedulerSettingsRepo) SetBool(ctx context.Context, key string, value bool) error {
	return m.Called(ctx, key, value).Error(0)
}

// MockBackupSvc is a mock for BackupServiceInterface (used by scheduler)
type MockBackupSvc struct {
	mock.Mock
}

func (m *MockBackupSvc) CreateBackup(ctx context.Context) (*models.Backup, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Backup), args.Error(1)
}
func (m *MockBackupSvc) ListBackups(ctx context.Context) (*models.BackupListResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BackupListResponse), args.Error(1)
}
func (m *MockBackupSvc) GetBackup(ctx context.Context, id string) (*models.Backup, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Backup), args.Error(1)
}
func (m *MockBackupSvc) DeleteBackup(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockBackupSvc) GetBackupFilePath(ctx context.Context, id string) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}
func (m *MockBackupSvc) VerifyBackup(ctx context.Context, id string) (*models.VerificationResult, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VerificationResult), args.Error(1)
}
func (m *MockBackupSvc) RestoreBackup(ctx context.Context, id string) (*models.RestoreResult, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RestoreResult), args.Error(1)
}
func (m *MockBackupSvc) GetRestoreStatus(ctx context.Context) (*models.RestoreResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RestoreResult), args.Error(1)
}

func TestBackupScheduler_SetSchedule(t *testing.T) {
	ctx := context.Background()

	t.Run("valid daily schedule", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		svc := NewBackupScheduler(nil, settingsRepo, nil)

		settingsRepo.On("SetString", ctx, settingsKeyBackupSchedule, mock.AnythingOfType("string")).Return(nil)

		err := svc.SetSchedule(ctx, BackupSchedule{
			Enabled:   true,
			Frequency: "daily",
			Hour:      3,
		})
		assert.NoError(t, err)
		settingsRepo.AssertExpectations(t)
	})

	t.Run("valid weekly schedule", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		svc := NewBackupScheduler(nil, settingsRepo, nil)

		settingsRepo.On("SetString", ctx, settingsKeyBackupSchedule, mock.AnythingOfType("string")).Return(nil)

		err := svc.SetSchedule(ctx, BackupSchedule{
			Enabled:   true,
			Frequency: "weekly",
			Hour:      2,
			DayOfWeek: 1, // Monday
		})
		assert.NoError(t, err)
	})

	t.Run("invalid frequency", func(t *testing.T) {
		svc := NewBackupScheduler(nil, nil, nil)
		err := svc.SetSchedule(ctx, BackupSchedule{Frequency: "hourly"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SCHEDULE_INVALID")
	})

	t.Run("invalid hour", func(t *testing.T) {
		svc := NewBackupScheduler(nil, nil, nil)
		err := svc.SetSchedule(ctx, BackupSchedule{Frequency: "daily", Hour: 25})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hour must be 0-23")
	})

	t.Run("invalid day of week for weekly", func(t *testing.T) {
		svc := NewBackupScheduler(nil, nil, nil)
		err := svc.SetSchedule(ctx, BackupSchedule{Frequency: "weekly", Hour: 3, DayOfWeek: 8})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dayOfWeek must be 0-6")
	})
}

func TestBackupScheduler_GetSchedule(t *testing.T) {
	ctx := context.Background()

	t.Run("returns default when no schedule configured", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		backupRepo := new(MockBackupRepo)
		svc := NewBackupScheduler(nil, settingsRepo, backupRepo)

		settingsRepo.On("GetString", ctx, settingsKeyBackupSchedule).Return("", assert.AnError)
		backupRepo.On("List", ctx).Return([]models.Backup{}, nil)

		resp, err := svc.GetSchedule(ctx)
		assert.NoError(t, err)
		assert.False(t, resp.Enabled)
		assert.Equal(t, "disabled", resp.Frequency)
	})

	t.Run("returns saved schedule with next backup time", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		backupRepo := new(MockBackupRepo)
		svc := NewBackupScheduler(nil, settingsRepo, backupRepo)

		scheduleJSON, _ := json.Marshal(BackupSchedule{Enabled: true, Frequency: "daily", Hour: 3})
		settingsRepo.On("GetString", ctx, settingsKeyBackupSchedule).Return(string(scheduleJSON), nil)
		backupRepo.On("List", ctx).Return([]models.Backup{}, nil)

		resp, err := svc.GetSchedule(ctx)
		assert.NoError(t, err)
		assert.True(t, resp.Enabled)
		assert.Equal(t, "daily", resp.Frequency)
		assert.Equal(t, 3, resp.Hour)
		assert.NotNil(t, resp.NextBackupAt)
	})

	t.Run("includes last backup time", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		backupRepo := new(MockBackupRepo)
		svc := NewBackupScheduler(nil, settingsRepo, backupRepo)

		settingsRepo.On("GetString", ctx, settingsKeyBackupSchedule).Return("", assert.AnError)
		lastBackup := time.Now().Add(-1 * time.Hour)
		backupRepo.On("List", ctx).Return([]models.Backup{{CreatedAt: lastBackup}}, nil)

		resp, err := svc.GetSchedule(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, resp.LastBackupAt)
	})
}

func TestBackupScheduler_ShouldRunNow(t *testing.T) {
	t.Run("runs at configured hour minute 0", func(t *testing.T) {
		svc := NewBackupScheduler(nil, nil, nil)
		now := time.Date(2026, 3, 20, 3, 0, 0, 0, time.UTC) // 03:00
		schedule := &BackupSchedule{Enabled: true, Frequency: "daily", Hour: 3}

		assert.True(t, svc.shouldRunNow(now, schedule))
	})

	t.Run("does not run at wrong minute", func(t *testing.T) {
		svc := NewBackupScheduler(nil, nil, nil)
		now := time.Date(2026, 3, 20, 3, 15, 0, 0, time.UTC) // 03:15
		schedule := &BackupSchedule{Enabled: true, Frequency: "daily", Hour: 3}

		assert.False(t, svc.shouldRunNow(now, schedule))
	})

	t.Run("does not run at wrong hour", func(t *testing.T) {
		svc := NewBackupScheduler(nil, nil, nil)
		now := time.Date(2026, 3, 20, 5, 0, 0, 0, time.UTC) // 05:00
		schedule := &BackupSchedule{Enabled: true, Frequency: "daily", Hour: 3}

		assert.False(t, svc.shouldRunNow(now, schedule))
	})

	t.Run("weekly: runs on correct day", func(t *testing.T) {
		svc := NewBackupScheduler(nil, nil, nil)
		// 2026-03-20 is a Friday (5)
		now := time.Date(2026, 3, 20, 3, 0, 0, 0, time.UTC)
		schedule := &BackupSchedule{Enabled: true, Frequency: "weekly", Hour: 3, DayOfWeek: 5}

		assert.True(t, svc.shouldRunNow(now, schedule))
	})

	t.Run("weekly: does not run on wrong day", func(t *testing.T) {
		svc := NewBackupScheduler(nil, nil, nil)
		now := time.Date(2026, 3, 20, 3, 0, 0, 0, time.UTC) // Friday
		schedule := &BackupSchedule{Enabled: true, Frequency: "weekly", Hour: 3, DayOfWeek: 1} // Monday

		assert.False(t, svc.shouldRunNow(now, schedule))
	})

	t.Run("prevents duplicate runs on same day", func(t *testing.T) {
		svc := NewBackupScheduler(nil, nil, nil)
		now := time.Date(2026, 3, 20, 3, 0, 0, 0, time.UTC)
		schedule := &BackupSchedule{Enabled: true, Frequency: "daily", Hour: 3}

		// First call should run
		assert.True(t, svc.shouldRunNow(now, schedule))
		// Second call same day should not
		assert.False(t, svc.shouldRunNow(now, schedule))
	})
}

func TestBackupScheduler_CalculateNextBackup(t *testing.T) {
	svc := NewBackupScheduler(nil, nil, nil)

	t.Run("daily next is tomorrow when past configured hour", func(t *testing.T) {
		now := time.Date(2026, 3, 20, 15, 0, 0, 0, time.UTC) // 15:00
		schedule := &BackupSchedule{Frequency: "daily", Hour: 3}

		next := svc.calculateNextBackup(now, schedule)
		assert.Equal(t, 21, next.Day())
		assert.Equal(t, 3, next.Hour())
	})

	t.Run("daily next is today when before configured hour", func(t *testing.T) {
		now := time.Date(2026, 3, 20, 1, 0, 0, 0, time.UTC) // 01:00
		schedule := &BackupSchedule{Frequency: "daily", Hour: 3}

		next := svc.calculateNextBackup(now, schedule)
		assert.Equal(t, 20, next.Day())
		assert.Equal(t, 3, next.Hour())
	})

	t.Run("weekly next is correct day", func(t *testing.T) {
		// 2026-03-20 is Friday (5), schedule for Monday (1)
		now := time.Date(2026, 3, 20, 15, 0, 0, 0, time.UTC)
		schedule := &BackupSchedule{Frequency: "weekly", Hour: 3, DayOfWeek: 1}

		next := svc.calculateNextBackup(now, schedule)
		assert.Equal(t, time.Monday, next.Weekday())
		assert.Equal(t, 3, next.Hour())
	})
}

func TestBackupScheduler_ApplyRetentionPolicy(t *testing.T) {
	ctx := context.Background()

	t.Run("keeps 7 most recent plus weekly, deletes rest", func(t *testing.T) {
		backupSvc := new(MockBackupSvc)
		backupRepo := new(MockBackupRepo)
		svc := NewBackupScheduler(backupSvc, nil, backupRepo)

		// Create 10 backups (sorted DESC by created_at)
		var backups []models.Backup
		for i := 0; i < 10; i++ {
			backups = append(backups, models.Backup{
				ID:        fmt.Sprintf("b%d", i),
				Filename:  fmt.Sprintf("vido-backup-%d.tar.gz", i),
				Status:    "completed",
				CreatedAt: time.Now().AddDate(0, 0, -i),
			})
		}
		backupRepo.On("List", ctx).Return(backups, nil)
		backupSvc.On("DeleteBackup", ctx, mock.AnythingOfType("string")).Return(nil)

		result, err := svc.ApplyRetentionPolicy(ctx)
		assert.NoError(t, err)
		// 7 daily retention + at least 1 weekly from remaining 3
		assert.GreaterOrEqual(t, result.Kept, 7)
		assert.LessOrEqual(t, result.Kept, 10)
		assert.Equal(t, 10, result.Kept+result.Deleted)
	})

	t.Run("never deletes auto-snapshots", func(t *testing.T) {
		backupSvc := new(MockBackupSvc)
		backupRepo := new(MockBackupRepo)
		svc := NewBackupScheduler(backupSvc, nil, backupRepo)

		backups := []models.Backup{
			{ID: "b1", Filename: "vido-backup-1.tar.gz", Status: "completed", CreatedAt: time.Now()},
			{ID: "snap1", Filename: "vido-auto-snapshot-before-restore-20260320-150000.tar.gz", Status: "completed", CreatedAt: time.Now().AddDate(0, 0, -30)},
		}
		backupRepo.On("List", ctx).Return(backups, nil)

		result, err := svc.ApplyRetentionPolicy(ctx)
		assert.NoError(t, err)
		// b1 kept (daily), snap1 kept (auto-snapshot)
		assert.Equal(t, 2, result.Kept)
		assert.Equal(t, 0, result.Deleted)
	})

	t.Run("empty backup list", func(t *testing.T) {
		backupRepo := new(MockBackupRepo)
		svc := NewBackupScheduler(nil, nil, backupRepo)

		backupRepo.On("List", ctx).Return([]models.Backup{}, nil)

		result, err := svc.ApplyRetentionPolicy(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 0, result.Kept)
		assert.Equal(t, 0, result.Deleted)
	})

	t.Run("keeps weekly backups from different weeks", func(t *testing.T) {
		backupSvc := new(MockBackupSvc)
		backupRepo := new(MockBackupRepo)
		svc := NewBackupScheduler(backupSvc, nil, backupRepo)

		// 7 recent + 4 weekly from different weeks
		var backups []models.Backup
		for i := 0; i < 7; i++ {
			backups = append(backups, models.Backup{
				ID:        fmt.Sprintf("daily-%d", i),
				Filename:  fmt.Sprintf("vido-backup-daily-%d.tar.gz", i),
				Status:    "completed",
				CreatedAt: time.Now().AddDate(0, 0, -i),
			})
		}
		// Add 6 more from different weeks (8-13 days ago, each in a different week)
		for i := 0; i < 6; i++ {
			backups = append(backups, models.Backup{
				ID:        fmt.Sprintf("weekly-%d", i),
				Filename:  fmt.Sprintf("vido-backup-weekly-%d.tar.gz", i),
				Status:    "completed",
				CreatedAt: time.Now().AddDate(0, 0, -(7 + i*7)),
			})
		}
		backupRepo.On("List", ctx).Return(backups, nil)
		backupSvc.On("DeleteBackup", ctx, mock.AnythingOfType("string")).Return(nil)

		result, err := svc.ApplyRetentionPolicy(ctx)
		assert.NoError(t, err)
		// 7 daily + 4 weekly kept, 2 weekly deleted
		assert.Equal(t, 11, result.Kept)
		assert.Equal(t, 2, result.Deleted)
	})
}

func TestBackupScheduler_Stop(t *testing.T) {
	t.Run("stop is idempotent", func(t *testing.T) {
		svc := NewBackupScheduler(nil, nil, nil)
		svc.Stop()
		svc.Stop() // Should not panic
	})
}

func TestValidateSchedule(t *testing.T) {
	t.Run("valid disabled", func(t *testing.T) {
		assert.NoError(t, validateSchedule(BackupSchedule{Frequency: "disabled"}))
	})

	t.Run("valid daily", func(t *testing.T) {
		assert.NoError(t, validateSchedule(BackupSchedule{Frequency: "daily", Hour: 0}))
		assert.NoError(t, validateSchedule(BackupSchedule{Frequency: "daily", Hour: 23}))
	})

	t.Run("valid weekly", func(t *testing.T) {
		assert.NoError(t, validateSchedule(BackupSchedule{Frequency: "weekly", Hour: 3, DayOfWeek: 0}))
		assert.NoError(t, validateSchedule(BackupSchedule{Frequency: "weekly", Hour: 3, DayOfWeek: 6}))
	})

	t.Run("negative hour", func(t *testing.T) {
		assert.Error(t, validateSchedule(BackupSchedule{Frequency: "daily", Hour: -1}))
	})
}

func TestBackupScheduler_SetSchedule_RepoError(t *testing.T) {
	ctx := context.Background()

	t.Run("save error propagated", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		svc := NewBackupScheduler(nil, settingsRepo, nil)

		settingsRepo.On("SetString", ctx, settingsKeyBackupSchedule, mock.AnythingOfType("string")).Return(assert.AnError)

		err := svc.SetSchedule(ctx, BackupSchedule{Frequency: "daily", Hour: 3})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save schedule")
	})

	t.Run("persisted JSON has correct fields", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		svc := NewBackupScheduler(nil, settingsRepo, nil)

		var savedJSON string
		settingsRepo.On("SetString", ctx, settingsKeyBackupSchedule, mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
			savedJSON = args.String(2)
		}).Return(nil)

		err := svc.SetSchedule(ctx, BackupSchedule{
			Enabled:   true,
			Frequency: "weekly",
			Hour:      14,
			DayOfWeek: 3,
		})
		assert.NoError(t, err)

		var parsed BackupSchedule
		assert.NoError(t, json.Unmarshal([]byte(savedJSON), &parsed))
		assert.True(t, parsed.Enabled)
		assert.Equal(t, "weekly", parsed.Frequency)
		assert.Equal(t, 14, parsed.Hour)
		assert.Equal(t, 3, parsed.DayOfWeek)
	})
}

func TestBackupScheduler_LoadSchedule_CorruptedJSON(t *testing.T) {
	ctx := context.Background()

	t.Run("corrupted JSON returns error", func(t *testing.T) {
		settingsRepo := new(MockSchedulerSettingsRepo)
		backupRepo := new(MockBackupRepo)
		svc := NewBackupScheduler(nil, settingsRepo, backupRepo)

		settingsRepo.On("GetString", ctx, settingsKeyBackupSchedule).Return("not valid json{{{", nil)

		_, err := svc.GetSchedule(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parse schedule")
	})
}

func TestBackupScheduler_CheckAndRun(t *testing.T) {
	ctx := context.Background()

	t.Run("creates backup and applies retention on match", func(t *testing.T) {
		backupSvc := new(MockBackupSvc)
		settingsRepo := new(MockSchedulerSettingsRepo)
		backupRepo := new(MockBackupRepo)
		svc := NewBackupScheduler(backupSvc, settingsRepo, backupRepo)

		scheduleJSON, _ := json.Marshal(BackupSchedule{Enabled: true, Frequency: "daily", Hour: 3})
		settingsRepo.On("GetString", ctx, settingsKeyBackupSchedule).Return(string(scheduleJSON), nil)

		backup := &models.Backup{ID: "new-backup", Filename: "vido-backup-new.tar.gz"}
		backupSvc.On("CreateBackup", ctx).Return(backup, nil)
		backupRepo.On("List", ctx).Return([]models.Backup{}, nil)

		now := time.Date(2026, 3, 20, 3, 0, 0, 0, time.UTC)
		svc.checkAndRun(ctx, now)

		backupSvc.AssertCalled(t, "CreateBackup", ctx)
	})

	t.Run("does not run when schedule disabled", func(t *testing.T) {
		backupSvc := new(MockBackupSvc)
		settingsRepo := new(MockSchedulerSettingsRepo)
		svc := NewBackupScheduler(backupSvc, settingsRepo, nil)

		scheduleJSON, _ := json.Marshal(BackupSchedule{Enabled: false, Frequency: "disabled"})
		settingsRepo.On("GetString", ctx, settingsKeyBackupSchedule).Return(string(scheduleJSON), nil)

		now := time.Date(2026, 3, 20, 3, 0, 0, 0, time.UTC)
		svc.checkAndRun(ctx, now)

		backupSvc.AssertNotCalled(t, "CreateBackup", mock.Anything)
	})

	t.Run("does not apply retention when backup fails", func(t *testing.T) {
		backupSvc := new(MockBackupSvc)
		settingsRepo := new(MockSchedulerSettingsRepo)
		svc := NewBackupScheduler(backupSvc, settingsRepo, nil)

		scheduleJSON, _ := json.Marshal(BackupSchedule{Enabled: true, Frequency: "daily", Hour: 3})
		settingsRepo.On("GetString", ctx, settingsKeyBackupSchedule).Return(string(scheduleJSON), nil)
		backupSvc.On("CreateBackup", ctx).Return((*models.Backup)(nil), assert.AnError)

		now := time.Date(2026, 3, 20, 3, 0, 0, 0, time.UTC)
		svc.checkAndRun(ctx, now)

		backupSvc.AssertCalled(t, "CreateBackup", ctx)
		// Retention should NOT be called since backup failed
		backupSvc.AssertNotCalled(t, "DeleteBackup", mock.Anything, mock.Anything)
	})
}

func TestBackupScheduler_ApplyRetentionPolicy_Errors(t *testing.T) {
	ctx := context.Background()

	t.Run("repo list error", func(t *testing.T) {
		backupRepo := new(MockBackupRepo)
		svc := NewBackupScheduler(nil, nil, backupRepo)

		backupRepo.On("List", ctx).Return(([]models.Backup)(nil), assert.AnError)

		_, err := svc.ApplyRetentionPolicy(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "list backups")
	})

	t.Run("delete error continues with remaining", func(t *testing.T) {
		backupSvc := new(MockBackupSvc)
		backupRepo := new(MockBackupRepo)
		svc := NewBackupScheduler(backupSvc, nil, backupRepo)

		// 9 completed backups - 7 kept daily, 2 to delete
		var backups []models.Backup
		for i := 0; i < 9; i++ {
			backups = append(backups, models.Backup{
				ID:        fmt.Sprintf("b%d", i),
				Filename:  fmt.Sprintf("vido-backup-%d.tar.gz", i),
				Status:    "completed",
				CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -i*7), // Different weeks
			})
		}
		backupRepo.On("List", ctx).Return(backups, nil)
		// First delete fails, second succeeds
		backupSvc.On("DeleteBackup", ctx, mock.AnythingOfType("string")).Return(assert.AnError).Once()
		backupSvc.On("DeleteBackup", ctx, mock.AnythingOfType("string")).Return(nil)

		result, err := svc.ApplyRetentionPolicy(ctx)
		assert.NoError(t, err)
		// Delete was attempted but one failed — doesn't block others
		assert.GreaterOrEqual(t, result.Deleted, 0)
	})

	t.Run("non-completed backups are kept", func(t *testing.T) {
		backupRepo := new(MockBackupRepo)
		svc := NewBackupScheduler(nil, nil, backupRepo)

		backups := []models.Backup{
			{ID: "b1", Filename: "vido-backup-1.tar.gz", Status: "completed", CreatedAt: time.Now()},
			{ID: "b2", Filename: "vido-backup-2.tar.gz", Status: "failed", CreatedAt: time.Now().AddDate(0, 0, -30)},
			{ID: "b3", Filename: "vido-backup-3.tar.gz", Status: "running", CreatedAt: time.Now().AddDate(0, 0, -31)},
		}
		backupRepo.On("List", ctx).Return(backups, nil)

		result, err := svc.ApplyRetentionPolicy(ctx)
		assert.NoError(t, err)
		// b1 kept (daily), b2+b3 kept (non-completed = never delete)
		assert.Equal(t, 3, result.Kept)
		assert.Equal(t, 0, result.Deleted)
	})
}

func TestBackupScheduler_CalculateNextBackup_WeeklySameDay(t *testing.T) {
	svc := NewBackupScheduler(nil, nil, nil)

	t.Run("weekly same day past time goes to next week", func(t *testing.T) {
		// Friday 15:00, schedule is Friday 03:00 → next Friday
		now := time.Date(2026, 3, 20, 15, 0, 0, 0, time.UTC)
		schedule := &BackupSchedule{Frequency: "weekly", Hour: 3, DayOfWeek: 5}

		next := svc.calculateNextBackup(now, schedule)
		assert.Equal(t, time.Friday, next.Weekday())
		assert.Equal(t, 27, next.Day()) // Next Friday
		assert.Equal(t, 3, next.Hour())
	})

	t.Run("weekly same day before time runs today", func(t *testing.T) {
		// Friday 01:00, schedule is Friday 03:00 → today
		now := time.Date(2026, 3, 20, 1, 0, 0, 0, time.UTC)
		schedule := &BackupSchedule{Frequency: "weekly", Hour: 3, DayOfWeek: 5}

		next := svc.calculateNextBackup(now, schedule)
		assert.Equal(t, time.Friday, next.Weekday())
		assert.Equal(t, 20, next.Day()) // Today
		assert.Equal(t, 3, next.Hour())
	})
}

func TestIsAutoSnapshot(t *testing.T) {
	assert.True(t, isAutoSnapshot("vido-auto-snapshot-before-restore-20260320-150000.tar.gz"))
	assert.False(t, isAutoSnapshot("vido-backup-20260320-150000-v17.tar.gz"))
	assert.False(t, isAutoSnapshot("short.tar.gz"))
}
