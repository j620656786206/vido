package models

import "time"

// BackupStatus represents the current state of a backup operation
type BackupStatus string

const (
	BackupStatusPending   BackupStatus = "pending"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusFailed    BackupStatus = "failed"
	BackupStatusCorrupted BackupStatus = "corrupted"
)

// Backup represents a database backup record
type Backup struct {
	ID            string       `json:"id"`
	Filename      string       `json:"filename"`
	SizeBytes     int64        `json:"size_bytes"`
	SchemaVersion int64        `json:"schema_version"`
	Checksum      string       `json:"checksum"`
	Status        BackupStatus `json:"status"`
	ErrorMessage  string       `json:"error_message,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
}

// BackupListResponse wraps the backup list with total size
type BackupListResponse struct {
	Backups        []Backup `json:"backups"`
	TotalSizeBytes int64    `json:"total_size_bytes"`
}

// VerificationStatus represents the result of a backup integrity check
type VerificationStatus string

const (
	VerificationStatusVerified  VerificationStatus = "verified"
	VerificationStatusCorrupted VerificationStatus = "corrupted"
	VerificationStatusMissing   VerificationStatus = "missing"
)

// RestoreStatus represents the current state of a restore operation
type RestoreStatus string

const (
	RestoreStatusInProgress RestoreStatus = "in_progress"
	RestoreStatusCompleted  RestoreStatus = "completed"
	RestoreStatusFailed     RestoreStatus = "failed"
)

// RestoreResult contains the outcome of a restore operation
type RestoreResult struct {
	RestoreID  string        `json:"restore_id"`
	Status     RestoreStatus `json:"status"`
	SnapshotID string        `json:"snapshot_id"`
	Message    string        `json:"message"`
	Error      string        `json:"error,omitempty"`
}

// VerificationResult contains the outcome of a backup verification
type VerificationResult struct {
	BackupID           string             `json:"backup_id"`
	Status             VerificationStatus `json:"status"`
	StoredChecksum     string             `json:"stored_checksum"`
	CalculatedChecksum string             `json:"calculated_checksum"`
	Match              bool               `json:"match"`
	VerifiedAt         time.Time          `json:"verified_at"`
}
