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

// BackupServiceInterface defines the contract for backup operations
type BackupServiceInterface interface {
	CreateBackup(ctx context.Context) (*models.Backup, error)
	ListBackups(ctx context.Context) (*models.BackupListResponse, error)
	GetBackup(ctx context.Context, id string) (*models.Backup, error)
	DeleteBackup(ctx context.Context, id string) error
	GetBackupFilePath(ctx context.Context, id string) (string, error)
}

// BackupService manages database backup operations
type BackupService struct {
	db            *sql.DB
	repo          repository.BackupRepositoryInterface
	backupDir     string
	schemaVersion int64
	mu            sync.Mutex
	running       bool
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

	// Step 5: Update backup record
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

	// Remove file
	filePath := filepath.Join(s.backupDir, backup.Filename)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		slog.Warn("Failed to remove backup file", "path", filePath, "error", err)
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
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

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
