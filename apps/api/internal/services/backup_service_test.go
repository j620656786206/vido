package services

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// MockBackupRepo implements BackupRepositoryInterface for testing
type MockBackupRepo struct {
	mock.Mock
}

func (m *MockBackupRepo) Create(ctx context.Context, backup *models.Backup) error {
	args := m.Called(ctx, backup)
	return args.Error(0)
}

func (m *MockBackupRepo) List(ctx context.Context) ([]models.Backup, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Backup), args.Error(1)
}

func (m *MockBackupRepo) GetByID(ctx context.Context, id string) (*models.Backup, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Backup), args.Error(1)
}

func (m *MockBackupRepo) Update(ctx context.Context, backup *models.Backup) error {
	args := m.Called(ctx, backup)
	return args.Error(0)
}

func (m *MockBackupRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBackupRepo) TotalSizeBytes(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// openVidoTestDB opens a FILE database the way the app does (foreign keys on,
// WAL) and applies every migration up to maxVersion (0 = all). The old fixture
// was a single un-keyed test_data table, which is exactly why a restore that
// broke on every real database stayed green (bugfix-restore-fails-on-real-database).
func openVidoTestDB(t *testing.T, path string, maxVersion int64) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(on)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	for _, m := range migrations.GetAll() {
		if maxVersion == 0 || m.Version() <= maxVersion {
			require.NoError(t, runner.Register(m))
		}
	}
	require.NoError(t, runner.Up(context.Background()))
	return db
}

// latestMigrationVersion is the newest registered migration.
func latestMigrationVersion() int64 {
	var v int64
	for _, m := range migrations.GetAll() {
		if m.Version() > v {
			v = m.Version()
		}
	}
	return v
}

// createTestDB creates a fully migrated Vido database (plus the legacy
// test_data table the older tests assert on).
func createTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	tmpDir := t.TempDir()
	db := openVidoTestDB(t, filepath.Join(tmpDir, "test.db"), 0)

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS test_data (id TEXT PRIMARY KEY, name TEXT)`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO test_data (id, name) VALUES ('1', 'test')`)
	require.NoError(t, err)

	return db, tmpDir
}

// createTestBackupArchive creates a valid tar.gz backup for testing
func createTestBackupArchive(t *testing.T, dir string, schemaVersion int64, dbData string) (string, string) {
	t.Helper()

	// A real Vido database at `schemaVersion`: migrated up to it when it is
	// older than the code, or fully migrated plus a fake newer row when it is
	// newer (a backup made by a newer Vido).
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "src.db")
	latest := latestMigrationVersion()
	upTo := schemaVersion
	if upTo >= latest {
		upTo = 0
	}
	testDB := openVidoTestDB(t, srcPath, upTo)
	if schemaVersion > latest {
		_, err := testDB.Exec(`INSERT INTO schema_migrations (version, name) VALUES (?, 'from_the_future')`, schemaVersion)
		require.NoError(t, err)
	}
	_, err := testDB.Exec(`CREATE TABLE test_data (id TEXT PRIMARY KEY, name TEXT)`)
	require.NoError(t, err)
	if dbData != "" {
		_, err = testDB.Exec(`INSERT INTO test_data (id, name) VALUES ('restored', ?)`, dbData)
		require.NoError(t, err)
	}
	// Pack a single self-contained file, as the app does (VACUUM INTO).
	tmpDBPath := filepath.Join(dir, "temp.db")
	_, err = testDB.Exec(fmt.Sprintf("VACUUM INTO '%s'", tmpDBPath))
	require.NoError(t, err)
	testDB.Close()

	// Create manifest
	manifest := backupManifest{
		SchemaVersion: schemaVersion,
		CreatedAt:     time.Now().Format(time.RFC3339),
		AppVersion:    "1.0.0",
	}
	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)

	// Create tar.gz archive
	filename := "vido-backup-test.tar.gz"
	archivePath := filepath.Join(dir, filename)
	outFile, err := os.Create(archivePath)
	require.NoError(t, err)

	hasher := sha256.New()
	multiWriter := io.MultiWriter(outFile, hasher)
	gzWriter := gzip.NewWriter(multiWriter)
	tarWriter := tar.NewWriter(gzWriter)

	// Add database file
	dbFile, err := os.Open(tmpDBPath)
	require.NoError(t, err)
	dbStat, err := dbFile.Stat()
	require.NoError(t, err)
	err = tarWriter.WriteHeader(&tar.Header{Name: "vido.db", Size: dbStat.Size(), Mode: 0o644, ModTime: time.Now()})
	require.NoError(t, err)
	_, err = io.Copy(tarWriter, dbFile)
	require.NoError(t, err)
	dbFile.Close()

	// Add manifest
	err = tarWriter.WriteHeader(&tar.Header{Name: "manifest.json", Size: int64(len(manifestJSON)), Mode: 0o644, ModTime: time.Now()})
	require.NoError(t, err)
	_, err = tarWriter.Write(manifestJSON)
	require.NoError(t, err)

	tarWriter.Close()
	gzWriter.Close()
	outFile.Close()
	os.Remove(tmpDBPath)

	checksum := hex.EncodeToString(hasher.Sum(nil))
	return filename, checksum
}

func TestBackupService_RestoreBackup(t *testing.T) {
	ctx := context.Background()

	t.Run("backup not found", func(t *testing.T) {
		repo := new(MockBackupRepo)
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, t.TempDir())

		repo.On("GetByID", ctx, "nonexistent").Return(nil, nil)

		_, err := svc.RestoreBackup(ctx, "nonexistent")
		assert.ErrorIs(t, err, ErrBackupNotFound)
	})

	t.Run("cannot restore non-completed backup", func(t *testing.T) {
		repo := new(MockBackupRepo)
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, t.TempDir())

		backup := &models.Backup{ID: "b1", Status: models.BackupStatusFailed}
		repo.On("GetByID", ctx, "b1").Return(backup, nil)

		_, err := svc.RestoreBackup(ctx, "b1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only completed backups")
	})

	t.Run("verify fails - integrity check mismatch", func(t *testing.T) {
		repo := new(MockBackupRepo)
		backupDir := t.TempDir()
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, backupDir)

		// Create an archive but with wrong checksum
		filename, _ := createTestBackupArchive(t, backupDir, 17, "data")

		backup := &models.Backup{
			ID:       "b1",
			Filename: filename,
			Checksum: "wrong-checksum",
			Status:   models.BackupStatusCompleted,
		}
		// GetByID called twice: once by RestoreBackup, once by VerifyBackup
		repo.On("GetByID", ctx, "b1").Return(backup, nil)
		// VerifyBackup marks backup as corrupted on mismatch
		repo.On("Update", ctx, mock.AnythingOfType("*models.Backup")).Return(nil)

		result, err := svc.RestoreBackup(ctx, "b1")
		assert.NoError(t, err)
		assert.Equal(t, models.RestoreStatusFailed, result.Status)
		assert.Contains(t, result.Error, "RESTORE_VERIFY_FAILED")
	})

	t.Run("incompatible schema version - backup newer", func(t *testing.T) {
		repo := new(MockBackupRepo)
		backupDir := t.TempDir()
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, backupDir)

		// Create archive with newer schema version (99)
		filename, checksum := createTestBackupArchive(t, backupDir, 99, "data")

		backup := &models.Backup{
			ID:       "b1",
			Filename: filename,
			Checksum: checksum,
			Status:   models.BackupStatusCompleted,
		}
		repo.On("GetByID", ctx, "b1").Return(backup, nil)
		// Snapshot lookup during rollback uses UUID - allow any ID
		repo.On("GetByID", ctx, mock.AnythingOfType("string")).Return((*models.Backup)(nil), nil).Maybe()
		// Mock for auto-snapshot creation
		repo.On("Create", ctx, mock.AnythingOfType("*models.Backup")).Return(nil)
		repo.On("Update", ctx, mock.AnythingOfType("*models.Backup")).Return(nil)

		result, err := svc.RestoreBackup(ctx, "b1")
		assert.NoError(t, err)
		assert.Equal(t, models.RestoreStatusFailed, result.Status)
		assert.Contains(t, result.Error, "RESTORE_INCOMPATIBLE_VERSION")
	})

	t.Run("successful restore", func(t *testing.T) {
		repo := new(MockBackupRepo)
		backupDir := t.TempDir()
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, backupDir)

		filename, checksum := createTestBackupArchive(t, backupDir, 17, "restored-data")

		backup := &models.Backup{
			ID:       "b1",
			Filename: filename,
			Checksum: checksum,
			Status:   models.BackupStatusCompleted,
		}
		repo.On("GetByID", ctx, "b1").Return(backup, nil)
		// Use MatchedBy for snapshot UUID lookups (not "b1")
		repo.On("GetByID", ctx, mock.MatchedBy(func(id string) bool {
			return id != "b1"
		})).Return((*models.Backup)(nil), nil).Maybe()
		repo.On("Create", ctx, mock.AnythingOfType("*models.Backup")).Return(nil)
		repo.On("Update", ctx, mock.AnythingOfType("*models.Backup")).Return(nil)

		result, err := svc.RestoreBackup(ctx, "b1")
		assert.NoError(t, err)
		if result.Status == models.RestoreStatusFailed {
			t.Logf("Restore failed with error: %s", result.Error)
		}
		assert.Equal(t, models.RestoreStatusCompleted, result.Status)
		assert.NotEmpty(t, result.SnapshotID)
		assert.Equal(t, "還原完成，資料庫已恢復", result.Message)
	})

	t.Run("concurrent restore rejected", func(t *testing.T) {
		repo := new(MockBackupRepo)
		backupDir := t.TempDir()
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, backupDir)
		// Simulate a restore in progress
		svc.restoreMu.Lock()
		svc.restoring = true
		svc.restoreMu.Unlock()

		_, err := svc.RestoreBackup(ctx, "b1")
		assert.ErrorIs(t, err, ErrRestoreInProgress)

		svc.restoreMu.Lock()
		svc.restoring = false
		svc.restoreMu.Unlock()
	})
}

func TestBackupService_GetRestoreStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("no restore in progress", func(t *testing.T) {
		repo := new(MockBackupRepo)
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, t.TempDir())

		result, err := svc.GetRestoreStatus(ctx)
		assert.NoError(t, err)
		assert.Equal(t, models.RestoreStatusCompleted, result.Status)
	})

	t.Run("restore result stored", func(t *testing.T) {
		repo := new(MockBackupRepo)
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, t.TempDir())
		svc.restoreResult = &models.RestoreResult{
			RestoreID: "r1",
			Status:    models.RestoreStatusInProgress,
			Message:   "正在還原...",
		}

		result, err := svc.GetRestoreStatus(ctx)
		assert.NoError(t, err)
		assert.Equal(t, models.RestoreStatusInProgress, result.Status)
		assert.Equal(t, "r1", result.RestoreID)
	})
}

func TestBackupService_CreateAutoSnapshot(t *testing.T) {
	ctx := context.Background()

	t.Run("auto-snapshot naming convention", func(t *testing.T) {
		repo := new(MockBackupRepo)
		backupDir := t.TempDir()
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, backupDir)

		repo.On("Create", ctx, mock.AnythingOfType("*models.Backup")).Return(nil)
		repo.On("Update", ctx, mock.AnythingOfType("*models.Backup")).Return(nil)

		snapshot, err := svc.createAutoSnapshot(ctx)
		assert.NoError(t, err)
		assert.Contains(t, snapshot.Filename, "vido-auto-snapshot-before-restore-")
		assert.Equal(t, models.BackupStatusCompleted, snapshot.Status)
		assert.NotEmpty(t, snapshot.Checksum)
	})
}

func TestBackupService_ExtractTarGz(t *testing.T) {
	t.Run("extracts files correctly", func(t *testing.T) {
		backupDir := t.TempDir()
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, nil, backupDir)

		filename, _ := createTestBackupArchive(t, backupDir, 17, "test")

		destDir := t.TempDir()
		err := svc.extractTarGz(filepath.Join(backupDir, filename), destDir)
		assert.NoError(t, err)

		// Verify vido.db was extracted
		_, err = os.Stat(filepath.Join(destDir, "vido.db"))
		assert.NoError(t, err)

		// Verify manifest.json was extracted
		_, err = os.Stat(filepath.Join(destDir, "manifest.json"))
		assert.NoError(t, err)
	})
}

func TestBackupService_ReadManifest(t *testing.T) {
	t.Run("reads valid manifest", func(t *testing.T) {
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, nil, "")

		tmpFile, err := os.CreateTemp("", "manifest-*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		manifestData := `{"schema_version": 17, "created_at": "2026-01-01T00:00:00Z", "app_version": "1.0.0"}`
		_, err = tmpFile.WriteString(manifestData)
		require.NoError(t, err)
		tmpFile.Close()

		manifest, err := svc.readManifest(tmpFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, int64(17), manifest.SchemaVersion)
		assert.Equal(t, "1.0.0", manifest.AppVersion)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, nil, "")

		_, err := svc.readManifest("/nonexistent/path")
		assert.Error(t, err)
	})
}

func TestBackupService_RestoreBackup_RepoError(t *testing.T) {
	ctx := context.Background()

	t.Run("repo GetByID returns error", func(t *testing.T) {
		repo := new(MockBackupRepo)
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, t.TempDir())

		repo.On("GetByID", ctx, "b1").Return((*models.Backup)(nil), assert.AnError)

		_, err := svc.RestoreBackup(ctx, "b1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "get backup")
	})
}

func TestBackupService_RestoreBackup_OlderSchemaVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("older schema version - compatible restore", func(t *testing.T) {
		repo := new(MockBackupRepo)
		backupDir := t.TempDir()
		db, _ := createTestDB(t)
		defer db.Close()

		// Current version 17, backup version 10 (older = compatible)
		svc := NewBackupService(db, repo, backupDir)

		filename, checksum := createTestBackupArchive(t, backupDir, 10, "old-data")

		backup := &models.Backup{
			ID:       "b1",
			Filename: filename,
			Checksum: checksum,
			Status:   models.BackupStatusCompleted,
		}
		repo.On("GetByID", ctx, "b1").Return(backup, nil)
		repo.On("GetByID", ctx, mock.MatchedBy(func(id string) bool {
			return id != "b1"
		})).Return((*models.Backup)(nil), nil).Maybe()
		repo.On("Create", ctx, mock.AnythingOfType("*models.Backup")).Return(nil)
		repo.On("Update", ctx, mock.AnythingOfType("*models.Backup")).Return(nil)

		result, err := svc.RestoreBackup(ctx, "b1")
		assert.NoError(t, err)
		// Older schema version should still succeed (compatible)
		assert.Equal(t, models.RestoreStatusCompleted, result.Status)
		assert.NotEmpty(t, result.SnapshotID)
	})
}

func TestBackupService_RestoreBackup_VerifyDBContent(t *testing.T) {
	ctx := context.Background()

	t.Run("restored data is actually in the database", func(t *testing.T) {
		repo := new(MockBackupRepo)
		backupDir := t.TempDir()
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, backupDir)

		// Verify original data exists
		var origName string
		err := db.QueryRow("SELECT name FROM test_data WHERE id='1'").Scan(&origName)
		require.NoError(t, err)
		assert.Equal(t, "test", origName)

		// Create backup with different data
		filename, checksum := createTestBackupArchive(t, backupDir, 17, "verification-data")

		backup := &models.Backup{
			ID:       "b1",
			Filename: filename,
			Checksum: checksum,
			Status:   models.BackupStatusCompleted,
		}
		repo.On("GetByID", ctx, "b1").Return(backup, nil)
		repo.On("GetByID", ctx, mock.MatchedBy(func(id string) bool {
			return id != "b1"
		})).Return((*models.Backup)(nil), nil).Maybe()
		repo.On("Create", ctx, mock.AnythingOfType("*models.Backup")).Return(nil)
		repo.On("Update", ctx, mock.AnythingOfType("*models.Backup")).Return(nil)

		result, err := svc.RestoreBackup(ctx, "b1")
		assert.NoError(t, err)
		assert.Equal(t, models.RestoreStatusCompleted, result.Status)

		// Verify original data was replaced
		var restoredName string
		err = db.QueryRow("SELECT name FROM test_data WHERE id='restored'").Scan(&restoredName)
		require.NoError(t, err)
		assert.Equal(t, "verification-data", restoredName)
	})
}

func TestBackupService_RestoreBackup_SnapshotFailure(t *testing.T) {
	ctx := context.Background()

	t.Run("auto-snapshot creation fails", func(t *testing.T) {
		repo := new(MockBackupRepo)
		backupDir := t.TempDir()
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, backupDir)

		filename, checksum := createTestBackupArchive(t, backupDir, 17, "data")
		backup := &models.Backup{
			ID:       "b1",
			Filename: filename,
			Checksum: checksum,
			Status:   models.BackupStatusCompleted,
		}
		repo.On("GetByID", ctx, "b1").Return(backup, nil)
		// Fail the snapshot Create call
		repo.On("Create", ctx, mock.AnythingOfType("*models.Backup")).Return(assert.AnError)
		repo.On("Update", ctx, mock.AnythingOfType("*models.Backup")).Return(nil)

		result, err := svc.RestoreBackup(ctx, "b1")
		assert.NoError(t, err)
		assert.Equal(t, models.RestoreStatusFailed, result.Status)
		assert.Contains(t, result.Error, "RESTORE_SNAPSHOT_FAILED")
	})
}

func TestBackupService_RestoreBackup_RunningStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("cannot restore backup with running status", func(t *testing.T) {
		repo := new(MockBackupRepo)
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, t.TempDir())

		backup := &models.Backup{ID: "b1", Status: models.BackupStatusRunning}
		repo.On("GetByID", ctx, "b1").Return(backup, nil)

		_, err := svc.RestoreBackup(ctx, "b1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only completed backups")
	})
}

func TestBackupService_ReadManifest_InvalidJSON(t *testing.T) {
	t.Run("returns error for invalid JSON", func(t *testing.T) {
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, nil, "")

		tmpFile, err := os.CreateTemp("", "manifest-*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString("not valid json{{{")
		require.NoError(t, err)
		tmpFile.Close()

		_, err = svc.readManifest(tmpFile.Name())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parse manifest")
	})
}

func TestBackupService_ExtractTarGz_InvalidArchive(t *testing.T) {
	t.Run("returns error for non-gzip file", func(t *testing.T) {
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, nil, "")

		// Create a non-gzip file
		tmpFile, err := os.CreateTemp("", "invalid-*.tar.gz")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString("this is not a gzip file")
		require.NoError(t, err)
		tmpFile.Close()

		err = svc.extractTarGz(tmpFile.Name(), t.TempDir())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "create gzip reader")
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, nil, "")

		err := svc.extractTarGz("/nonexistent/file.tar.gz", t.TempDir())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "open archive")
	})
}

func TestBackupService_ListBackups(t *testing.T) {
	ctx := context.Background()

	t.Run("returns ErrDatabaseIncomplete when repo List returns ErrTableMissing", func(t *testing.T) {
		repo := new(MockBackupRepo)
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, t.TempDir())

		repo.On("List", ctx).Return(nil, fmt.Errorf("query backups: %w: no such table: backups", repository.ErrTableMissing))

		result, err := svc.ListBackups(ctx)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrDatabaseIncomplete)
	})

	t.Run("returns ErrDatabaseIncomplete when repo TotalSizeBytes returns ErrTableMissing", func(t *testing.T) {
		repo := new(MockBackupRepo)
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, t.TempDir())

		repo.On("List", ctx).Return([]models.Backup{}, nil)
		repo.On("TotalSizeBytes", ctx).Return(int64(0), fmt.Errorf("sum backup sizes: %w: no such table: backups", repository.ErrTableMissing))

		result, err := svc.ListBackups(ctx)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrDatabaseIncomplete)
	})

	t.Run("returns generic error for non-table-missing repo errors", func(t *testing.T) {
		repo := new(MockBackupRepo)
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, t.TempDir())

		repo.On("List", ctx).Return(nil, assert.AnError)

		result, err := svc.ListBackups(ctx)
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.NotErrorIs(t, err, ErrDatabaseIncomplete)
	})

	t.Run("returns valid response with empty backups", func(t *testing.T) {
		repo := new(MockBackupRepo)
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, t.TempDir())

		repo.On("List", ctx).Return([]models.Backup{}, nil)
		repo.On("TotalSizeBytes", ctx).Return(int64(0), nil)

		result, err := svc.ListBackups(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Backups)
		assert.Equal(t, int64(0), result.TotalSizeBytes)
	})

	t.Run("returns valid response with backups and total size", func(t *testing.T) {
		repo := new(MockBackupRepo)
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, t.TempDir())

		backups := []models.Backup{
			{ID: "b1", Filename: "backup-1.tar.gz", SizeBytes: 1024, Status: models.BackupStatusCompleted},
			{ID: "b2", Filename: "backup-2.tar.gz", SizeBytes: 2048, Status: models.BackupStatusCompleted},
		}
		repo.On("List", ctx).Return(backups, nil)
		repo.On("TotalSizeBytes", ctx).Return(int64(3072), nil)

		result, err := svc.ListBackups(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Backups, 2)
		assert.Equal(t, int64(3072), result.TotalSizeBytes)
	})

	t.Run("returns empty slice not nil when repo returns nil backups", func(t *testing.T) {
		repo := new(MockBackupRepo)
		db, _ := createTestDB(t)
		defer db.Close()

		svc := NewBackupService(db, repo, t.TempDir())

		repo.On("List", ctx).Return(nil, nil)
		repo.On("TotalSizeBytes", ctx).Return(int64(0), nil)

		result, err := svc.ListBackups(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Backups)
		assert.Empty(t, result.Backups)
	})
}

func TestIsSubPath(t *testing.T) {
	t.Run("valid sub path", func(t *testing.T) {
		assert.True(t, isSubPath("/base/dir", "/base/dir/sub/file.txt"))
	})

	t.Run("same path", func(t *testing.T) {
		assert.True(t, isSubPath("/base/dir", "/base/dir"))
	})

	t.Run("traversal attempt", func(t *testing.T) {
		assert.False(t, isSubPath("/base/dir", "/base/other"))
	})

	t.Run("prefix overlap - dir-extra is NOT subpath of dir", func(t *testing.T) {
		assert.False(t, isSubPath("/base/dir", "/base/dir-extra/file.txt"))
	})
}
