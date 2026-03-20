package services

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// ErrBackupInProgress is returned when a backup is already running
var ErrBackupInProgress = errors.New("backup in progress")

// ErrBackupNotFound is returned when a backup ID is not found
var ErrBackupNotFound = errors.New("backup not found")

// ErrRestoreInProgress is returned when a restore is already running
var ErrRestoreInProgress = errors.New("restore in progress")

// ErrIncompatibleVersion is returned when the backup schema version is newer than current
var ErrIncompatibleVersion = errors.New("incompatible schema version")

// BackupServiceInterface defines the contract for backup operations
type BackupServiceInterface interface {
	CreateBackup(ctx context.Context) (*models.Backup, error)
	ListBackups(ctx context.Context) (*models.BackupListResponse, error)
	GetBackup(ctx context.Context, id string) (*models.Backup, error)
	DeleteBackup(ctx context.Context, id string) error
	GetBackupFilePath(ctx context.Context, id string) (string, error)
	VerifyBackup(ctx context.Context, id string) (*models.VerificationResult, error)
	RestoreBackup(ctx context.Context, id string) (*models.RestoreResult, error)
	GetRestoreStatus(ctx context.Context) (*models.RestoreResult, error)
}

// BackupService manages database backup operations
type BackupService struct {
	db            *sql.DB
	repo          repository.BackupRepositoryInterface
	backupDir     string
	schemaVersion int64
	mu            sync.Mutex
	running       bool
	restoreMu     sync.Mutex
	restoring     bool
	restoreResult *models.RestoreResult
}

// Compile-time interface verification
var _ BackupServiceInterface = (*BackupService)(nil)

// NewBackupService creates a new BackupService
func NewBackupService(db *sql.DB, repo repository.BackupRepositoryInterface, backupDir string, schemaVersion int64) *BackupService {
	return &BackupService{
		db:            db,
		repo:          repo,
		backupDir:     backupDir,
		schemaVersion: schemaVersion,
	}
}

// CreateBackup creates an atomic database backup packaged as tar.gz
func (s *BackupService) CreateBackup(ctx context.Context) (*models.Backup, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, ErrBackupInProgress
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	now := time.Now()
	backupID := uuid.New().String()
	filename := fmt.Sprintf("vido-backup-%s-v%d.tar.gz", now.Format("20060102-150405"), s.schemaVersion)

	backup := &models.Backup{
		ID:            backupID,
		Filename:      filename,
		SchemaVersion: s.schemaVersion,
		Status:        models.BackupStatusRunning,
		CreatedAt:     now,
	}

	if err := s.repo.Create(ctx, backup); err != nil {
		return nil, fmt.Errorf("create backup record: %w", err)
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(s.backupDir, 0o755); err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("create backup dir: %v", err))
		return backup, fmt.Errorf("create backup dir: %w", err)
	}

	// Step 1: Create atomic SQLite backup to temp file
	tmpDBFile, err := os.CreateTemp("", "vido-backup-*.db")
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("create temp file: %v", err))
		return backup, fmt.Errorf("create temp file: %w", err)
	}
	tmpDBPath := tmpDBFile.Name()
	tmpDBFile.Close()
	defer os.Remove(tmpDBPath)

	if err := s.sqliteBackup(ctx, tmpDBPath); err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("sqlite backup: %v", err))
		return backup, fmt.Errorf("sqlite backup: %w", err)
	}

	// Step 2: Create manifest
	manifest := s.createManifest(now)

	// Step 3: Package into tar.gz
	finalPath := filepath.Join(s.backupDir, filename)
	tmpTarPath := finalPath + ".tmp"

	checksum, sizeBytes, err := s.createTarGz(tmpTarPath, tmpDBPath, manifest)
	if err != nil {
		os.Remove(tmpTarPath)
		s.failBackup(ctx, backup, fmt.Sprintf("create tar.gz: %v", err))
		return backup, fmt.Errorf("create tar.gz: %w", err)
	}

	// Step 4: Atomic move to final location
	if err := os.Rename(tmpTarPath, finalPath); err != nil {
		os.Remove(tmpTarPath)
		s.failBackup(ctx, backup, fmt.Sprintf("move backup: %v", err))
		return backup, fmt.Errorf("move backup: %w", err)
	}

	// Step 5: Write .sha256 sidecar file
	sha256Path := finalPath + ".sha256"
	if err := os.WriteFile(sha256Path, []byte(checksum+"  "+filename+"\n"), 0o644); err != nil {
		slog.Warn("Failed to write SHA-256 sidecar file", "path", sha256Path, "error", err)
	}

	// Step 6: Update backup record
	backup.SizeBytes = sizeBytes
	backup.Checksum = checksum
	backup.Status = models.BackupStatusCompleted
	if err := s.repo.Update(ctx, backup); err != nil {
		slog.Error("Failed to update backup record", "error", err, "id", backupID)
	}

	slog.Info("Backup completed", "id", backupID, "filename", filename, "size_bytes", sizeBytes)
	return backup, nil
}

// ListBackups returns all backups with total size
func (s *BackupService) ListBackups(ctx context.Context) (*models.BackupListResponse, error) {
	backups, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}

	totalSize, err := s.repo.TotalSizeBytes(ctx)
	if err != nil {
		return nil, fmt.Errorf("get total size: %w", err)
	}

	if backups == nil {
		backups = []models.Backup{}
	}

	return &models.BackupListResponse{
		Backups:        backups,
		TotalSizeBytes: totalSize,
	}, nil
}

// GetBackup retrieves a backup by ID
func (s *BackupService) GetBackup(ctx context.Context, id string) (*models.Backup, error) {
	backup, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get backup: %w", err)
	}
	if backup == nil {
		return nil, ErrBackupNotFound
	}
	return backup, nil
}

// DeleteBackup removes a backup record and its file
func (s *BackupService) DeleteBackup(ctx context.Context, id string) error {
	backup, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get backup: %w", err)
	}
	if backup == nil {
		return ErrBackupNotFound
	}

	// Remove backup file and .sha256 sidecar
	filePath := filepath.Join(s.backupDir, backup.Filename)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		slog.Warn("Failed to remove backup file", "path", filePath, "error", err)
	}
	sha256Path := filePath + ".sha256"
	if err := os.Remove(sha256Path); err != nil && !os.IsNotExist(err) {
		slog.Warn("Failed to remove SHA-256 sidecar", "path", sha256Path, "error", err)
	}

	// Remove record
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete backup record: %w", err)
	}

	slog.Info("Backup deleted", "id", id, "filename", backup.Filename)
	return nil
}

// GetBackupFilePath returns the file path for downloading a backup
func (s *BackupService) GetBackupFilePath(ctx context.Context, id string) (string, error) {
	backup, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get backup: %w", err)
	}
	if backup == nil {
		return "", ErrBackupNotFound
	}

	filePath := filepath.Join(s.backupDir, backup.Filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("backup file not found: %s", backup.Filename)
	}

	return filePath, nil
}

// sqliteBackup performs an atomic SQLite backup using the backup API
func (s *BackupService) sqliteBackup(ctx context.Context, destPath string) error {
	// Use SQLite VACUUM INTO for atomic backup (works with WAL mode)
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", destPath))
	if err != nil {
		return fmt.Errorf("vacuum into: %w", err)
	}
	return nil
}

type backupManifest struct {
	SchemaVersion int64  `json:"schemaVersion"`
	CreatedAt     string `json:"createdAt"`
	AppVersion    string `json:"appVersion"`
}

func (s *BackupService) createManifest(now time.Time) backupManifest {
	return backupManifest{
		SchemaVersion: s.schemaVersion,
		CreatedAt:     now.Format(time.RFC3339),
		AppVersion:    "1.0.0",
	}
}

func (s *BackupService) createTarGz(outputPath, dbPath string, manifest backupManifest) (checksum string, sizeBytes int64, err error) {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return "", 0, fmt.Errorf("create output file: %w", err)
	}
	defer outFile.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(outFile, hasher)

	gzWriter := gzip.NewWriter(multiWriter)
	tarWriter := tar.NewWriter(gzWriter)

	// Add database file
	if err := addFileToTar(tarWriter, dbPath, "vido.db"); err != nil {
		return "", 0, fmt.Errorf("add db to tar: %w", err)
	}

	// Add manifest
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", 0, fmt.Errorf("marshal manifest: %w", err)
	}
	if err := addBytesToTar(tarWriter, manifestJSON, "manifest.json"); err != nil {
		return "", 0, fmt.Errorf("add manifest to tar: %w", err)
	}

	// Close writers to flush
	if err := tarWriter.Close(); err != nil {
		return "", 0, fmt.Errorf("close tar: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return "", 0, fmt.Errorf("close gzip: %w", err)
	}

	// Get file size
	stat, err := outFile.Stat()
	if err != nil {
		return "", 0, fmt.Errorf("stat output: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), stat.Size(), nil
}

func addFileToTar(tw *tar.Writer, srcPath, nameInTar string) error {
	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    nameInTar,
		Size:    stat.Size(),
		Mode:    0o644,
		ModTime: stat.ModTime(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

func addBytesToTar(tw *tar.Writer, data []byte, nameInTar string) error {
	header := &tar.Header{
		Name:    nameInTar,
		Size:    int64(len(data)),
		Mode:    0o644,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

func (s *BackupService) failBackup(ctx context.Context, backup *models.Backup, errMsg string) {
	backup.Status = models.BackupStatusFailed
	backup.ErrorMessage = errMsg
	if err := s.repo.Update(ctx, backup); err != nil {
		slog.Error("Failed to update backup status to failed", "error", err, "id", backup.ID)
	}
}

// VerifyBackup recalculates the checksum of a backup file and compares with stored value
func (s *BackupService) VerifyBackup(ctx context.Context, id string) (*models.VerificationResult, error) {
	backup, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get backup: %w", err)
	}
	if backup == nil {
		return nil, ErrBackupNotFound
	}

	if backup.Status != models.BackupStatusCompleted {
		return nil, fmt.Errorf("cannot verify backup with status %q: only completed backups can be verified", backup.Status)
	}

	result := &models.VerificationResult{
		BackupID:       id,
		StoredChecksum: backup.Checksum,
		VerifiedAt:     time.Now(),
	}

	filePath := filepath.Join(s.backupDir, backup.Filename)
	calculatedChecksum, err := calculateFileChecksum(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Status = models.VerificationStatusMissing
			return result, nil
		}
		return nil, fmt.Errorf("calculate checksum: %w", err)
	}

	result.CalculatedChecksum = calculatedChecksum
	result.Match = calculatedChecksum == backup.Checksum

	if result.Match {
		result.Status = models.VerificationStatusVerified
	} else {
		result.Status = models.VerificationStatusCorrupted
		backup.Status = models.BackupStatusCorrupted
		backup.ErrorMessage = "Checksum mismatch detected"
		if err := s.repo.Update(ctx, backup); err != nil {
			slog.Error("Failed to update backup status to corrupted", "error", err, "id", id)
		}
	}

	slog.Info("Backup verification completed", "id", id, "status", result.Status)
	return result, nil
}

// RestoreBackup restores the database from a backup archive (Story 6-7)
func (s *BackupService) RestoreBackup(ctx context.Context, id string) (*models.RestoreResult, error) {
	s.restoreMu.Lock()
	if s.restoring {
		s.restoreMu.Unlock()
		return nil, ErrRestoreInProgress
	}
	s.restoring = true
	s.restoreMu.Unlock()
	defer func() {
		s.restoreMu.Lock()
		s.restoring = false
		s.restoreMu.Unlock()
	}()

	result := &models.RestoreResult{
		RestoreID: uuid.New().String(),
		Status:    models.RestoreStatusInProgress,
	}

	// Step 1: Get and validate the backup
	backup, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get backup: %w", err)
	}
	if backup == nil {
		return nil, ErrBackupNotFound
	}
	if backup.Status != models.BackupStatusCompleted {
		return nil, fmt.Errorf("cannot restore backup with status %q: only completed backups can be restored", backup.Status)
	}

	// Step 2: Verify backup integrity before restore
	verifyResult, err := s.VerifyBackup(ctx, id)
	if err != nil {
		result.Status = models.RestoreStatusFailed
		result.Error = fmt.Sprintf("RESTORE_VERIFY_FAILED: %v", err)
		s.setRestoreResult(result)
		return result, nil
	}
	if !verifyResult.Match {
		result.Status = models.RestoreStatusFailed
		result.Error = "RESTORE_VERIFY_FAILED: backup integrity check failed, checksum mismatch"
		s.setRestoreResult(result)
		return result, nil
	}

	// Step 3: Create auto-snapshot before restore (NFR-R9)
	snapshot, err := s.createAutoSnapshot(ctx)
	if err != nil {
		result.Status = models.RestoreStatusFailed
		result.Error = fmt.Sprintf("RESTORE_SNAPSHOT_FAILED: %v", err)
		s.setRestoreResult(result)
		return result, nil
	}
	result.SnapshotID = snapshot.ID
	result.Message = "自動快照已建立，正在還原..."
	s.setRestoreResult(result)

	// Step 4: Extract tar.gz to temp directory
	backupFilePath := filepath.Join(s.backupDir, backup.Filename)
	tmpDir, err := os.MkdirTemp("", "vido-restore-*")
	if err != nil {
		s.rollbackFromSnapshot(ctx, snapshot.ID, result)
		return result, nil
	}
	defer os.RemoveAll(tmpDir)

	if err := s.extractTarGz(backupFilePath, tmpDir); err != nil {
		result.Status = models.RestoreStatusFailed
		result.Error = fmt.Sprintf("RESTORE_EXTRACT_FAILED: %v", err)
		s.rollbackFromSnapshot(ctx, snapshot.ID, result)
		return result, nil
	}

	// Step 5: Read and check schema version compatibility
	manifestPath := filepath.Join(tmpDir, "manifest.json")
	manifest, err := s.readManifest(manifestPath)
	if err != nil {
		result.Status = models.RestoreStatusFailed
		result.Error = fmt.Sprintf("RESTORE_EXTRACT_FAILED: cannot read manifest: %v", err)
		s.rollbackFromSnapshot(ctx, snapshot.ID, result)
		return result, nil
	}

	if manifest.SchemaVersion > s.schemaVersion {
		result.Status = models.RestoreStatusFailed
		result.Error = fmt.Sprintf("RESTORE_INCOMPATIBLE_VERSION: backup schema version %d is newer than current %d", manifest.SchemaVersion, s.schemaVersion)
		s.setRestoreResult(result)
		return result, nil
	}

	// Step 6: Replace current database with backup database
	backupDBPath := filepath.Join(tmpDir, "vido.db")
	if _, err := os.Stat(backupDBPath); os.IsNotExist(err) {
		result.Status = models.RestoreStatusFailed
		result.Error = "RESTORE_EXTRACT_FAILED: backup archive does not contain vido.db"
		s.rollbackFromSnapshot(ctx, snapshot.ID, result)
		return result, nil
	}

	if err := s.replaceDatabase(ctx, backupDBPath); err != nil {
		result.Status = models.RestoreStatusFailed
		result.Error = fmt.Sprintf("RESTORE_DB_FAILED: %v", err)
		s.rollbackFromSnapshot(ctx, snapshot.ID, result)
		return result, nil
	}

	// Step 7: Success
	result.Status = models.RestoreStatusCompleted
	result.Message = "還原完成，資料庫已恢復"
	s.setRestoreResult(result)

	slog.Info("Restore completed successfully", "backup_id", id, "restore_id", result.RestoreID, "snapshot_id", snapshot.ID)
	return result, nil
}

// GetRestoreStatus returns the current restore operation status
func (s *BackupService) GetRestoreStatus(ctx context.Context) (*models.RestoreResult, error) {
	s.restoreMu.Lock()
	defer s.restoreMu.Unlock()

	if s.restoreResult == nil {
		return &models.RestoreResult{
			Status:  models.RestoreStatusCompleted,
			Message: "沒有進行中的還原作業",
		}, nil
	}
	return s.restoreResult, nil
}

func (s *BackupService) setRestoreResult(result *models.RestoreResult) {
	s.restoreMu.Lock()
	s.restoreResult = result
	s.restoreMu.Unlock()
}

// createAutoSnapshot creates an automatic backup before restore (NFR-R9)
func (s *BackupService) createAutoSnapshot(ctx context.Context) (*models.Backup, error) {
	now := time.Now()
	snapshotID := uuid.New().String()
	filename := fmt.Sprintf("vido-auto-snapshot-before-restore-%s.tar.gz", now.Format("20060102-150405"))

	snapshot := &models.Backup{
		ID:            snapshotID,
		Filename:      filename,
		SchemaVersion: s.schemaVersion,
		Status:        models.BackupStatusRunning,
		CreatedAt:     now,
	}

	if err := s.repo.Create(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("create snapshot record: %w", err)
	}

	if err := os.MkdirAll(s.backupDir, 0o755); err != nil {
		s.failBackup(ctx, snapshot, fmt.Sprintf("create backup dir: %v", err))
		return nil, fmt.Errorf("create backup dir: %w", err)
	}

	tmpDBFile, err := os.CreateTemp("", "vido-snapshot-*.db")
	if err != nil {
		s.failBackup(ctx, snapshot, fmt.Sprintf("create temp file: %v", err))
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpDBPath := tmpDBFile.Name()
	tmpDBFile.Close()
	defer os.Remove(tmpDBPath)

	if err := s.sqliteBackup(ctx, tmpDBPath); err != nil {
		s.failBackup(ctx, snapshot, fmt.Sprintf("sqlite backup: %v", err))
		return nil, fmt.Errorf("sqlite snapshot: %w", err)
	}

	manifest := s.createManifest(now)
	finalPath := filepath.Join(s.backupDir, filename)
	tmpTarPath := finalPath + ".tmp"

	checksum, sizeBytes, err := s.createTarGz(tmpTarPath, tmpDBPath, manifest)
	if err != nil {
		os.Remove(tmpTarPath)
		s.failBackup(ctx, snapshot, fmt.Sprintf("create tar.gz: %v", err))
		return nil, fmt.Errorf("create snapshot archive: %w", err)
	}

	if err := os.Rename(tmpTarPath, finalPath); err != nil {
		os.Remove(tmpTarPath)
		s.failBackup(ctx, snapshot, fmt.Sprintf("move snapshot: %v", err))
		return nil, fmt.Errorf("move snapshot: %w", err)
	}

	snapshot.SizeBytes = sizeBytes
	snapshot.Checksum = checksum
	snapshot.Status = models.BackupStatusCompleted
	if err := s.repo.Update(ctx, snapshot); err != nil {
		slog.Error("Failed to update snapshot record", "error", err, "id", snapshotID)
	}

	slog.Info("Auto-snapshot created before restore", "id", snapshotID, "filename", filename)
	return snapshot, nil
}

// extractTarGz extracts a tar.gz archive to the destination directory
func (s *BackupService) extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		// Sanitize file name to prevent path traversal
		cleanName := filepath.Clean(header.Name)
		if filepath.IsAbs(cleanName) || cleanName == ".." || len(cleanName) > 0 && cleanName[0] == '/' {
			return fmt.Errorf("invalid tar entry name: %s", header.Name)
		}

		targetPath := filepath.Join(destDir, cleanName)
		// Verify the target path is within the destination directory
		if !isSubPath(destDir, targetPath) {
			return fmt.Errorf("tar entry %q escapes destination directory", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("create dir %s: %w", cleanName, err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("create file %s: %w", cleanName, err)
			}
			// Limit file size to 10GB to prevent decompression bombs
			limited := io.LimitReader(tarReader, 10*1024*1024*1024)
			if _, err := io.Copy(outFile, limited); err != nil {
				outFile.Close()
				return fmt.Errorf("extract file %s: %w", cleanName, err)
			}
			outFile.Close()
		}
	}
	return nil
}

// isSubPath checks if target is within the base directory
func isSubPath(base, target string) bool {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	return len(absTarget) >= len(absBase) && absTarget[:len(absBase)] == absBase
}

// readManifest reads and parses the backup manifest
func (s *BackupService) readManifest(path string) (*backupManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var manifest backupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &manifest, nil
}

// replaceDatabase replaces the current database with the backup database using SQLite backup API
func (s *BackupService) replaceDatabase(ctx context.Context, backupDBPath string) error {
	// Open the backup database
	backupDB, err := sql.Open("sqlite3", backupDBPath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("open backup db: %w", err)
	}
	defer backupDB.Close()

	// Verify backup database is valid
	if err := backupDB.PingContext(ctx); err != nil {
		return fmt.Errorf("validate backup db: %w", err)
	}

	// Use SQLite's built-in mechanism: load backup data into current DB
	// We do this by running VACUUM INTO to save current state, then
	// restore by overwriting tables from the backup
	// Using a transaction to replace all data atomically
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin restore transaction: %w", err)
	}
	defer tx.Rollback()

	// Attach the backup database
	_, err = tx.ExecContext(ctx, fmt.Sprintf("ATTACH DATABASE '%s' AS restore_db", backupDBPath))
	if err != nil {
		return fmt.Errorf("attach backup db: %w", err)
	}

	// Get list of tables from the backup database
	rows, err := tx.QueryContext(ctx, "SELECT name FROM restore_db.sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != 'schema_migrations'")
	if err != nil {
		return fmt.Errorf("list backup tables: %w", err)
	}

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return fmt.Errorf("scan table name: %w", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate backup tables: %w", err)
	}
	rows.Close()

	// For each table in backup: delete current data, copy from backup
	for _, table := range tables {
		// Check if table exists in current database
		var count int
		err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil || count == 0 {
			slog.Warn("Skipping restore table not in current schema", "table", table)
			continue
		}

		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM main.%q", table)); err != nil {
			return fmt.Errorf("clear table %s: %w", table, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO main.%q SELECT * FROM restore_db.%q", table, table)); err != nil {
			return fmt.Errorf("restore table %s: %w", table, err)
		}
	}

	// Commit first, then detach outside the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit restore: %w", err)
	}

	// Detach restore database outside the transaction
	if _, err := s.db.ExecContext(ctx, "DETACH DATABASE restore_db"); err != nil {
		slog.Warn("Failed to detach restore database", "error", err)
	}

	slog.Info("Database restored successfully", "tables_restored", len(tables))
	return nil
}

// rollbackFromSnapshot attempts to restore from the auto-snapshot when restore fails
func (s *BackupService) rollbackFromSnapshot(ctx context.Context, snapshotID string, result *models.RestoreResult) {
	snapshot, err := s.repo.GetByID(ctx, snapshotID)
	if err != nil || snapshot == nil {
		result.Error += " | RESTORE_ROLLBACK_FAILED: cannot find snapshot"
		s.setRestoreResult(result)
		slog.Error("Rollback failed: snapshot not found", "snapshot_id", snapshotID)
		return
	}

	snapshotPath := filepath.Join(s.backupDir, snapshot.Filename)
	tmpDir, err := os.MkdirTemp("", "vido-rollback-*")
	if err != nil {
		result.Error += " | RESTORE_ROLLBACK_FAILED: " + err.Error()
		s.setRestoreResult(result)
		return
	}
	defer os.RemoveAll(tmpDir)

	if err := s.extractTarGz(snapshotPath, tmpDir); err != nil {
		result.Error += " | RESTORE_ROLLBACK_FAILED: " + err.Error()
		s.setRestoreResult(result)
		return
	}

	dbPath := filepath.Join(tmpDir, "vido.db")
	if err := s.replaceDatabase(ctx, dbPath); err != nil {
		result.Error += " | RESTORE_ROLLBACK_FAILED: " + err.Error()
		s.setRestoreResult(result)
		slog.Error("Rollback failed", "error", err, "snapshot_id", snapshotID)
		return
	}

	result.Error += " | RESTORE_ROLLBACK_SUCCESS: recovered from auto-snapshot"
	s.setRestoreResult(result)
	slog.Info("Rollback from snapshot successful", "snapshot_id", snapshotID)
}

func calculateFileChecksum(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
