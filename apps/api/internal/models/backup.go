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
	SizeBytes     int64        `json:"sizeBytes"`
	SchemaVersion int64        `json:"schemaVersion"`
	Checksum      string       `json:"checksum"`
	Status        BackupStatus `json:"status"`
	ErrorMessage  string       `json:"errorMessage,omitempty"`
	CreatedAt     time.Time    `json:"createdAt"`
}

// BackupListResponse wraps the backup list with total size
type BackupListResponse struct {
	Backups        []Backup `json:"backups"`
	TotalSizeBytes int64    `json:"totalSizeBytes"`
}

// VerificationStatus represents the result of a backup integrity check
type VerificationStatus string

const (
	VerificationStatusVerified  VerificationStatus = "verified"
	VerificationStatusCorrupted VerificationStatus = "corrupted"
	VerificationStatusMissing   VerificationStatus = "missing"
)

// VerificationResult contains the outcome of a backup verification
type VerificationResult struct {
	BackupID           string             `json:"backupId"`
	Status             VerificationStatus `json:"status"`
	StoredChecksum     string             `json:"storedChecksum"`
	CalculatedChecksum string             `json:"calculatedChecksum"`
	Match              bool               `json:"match"`
	VerifiedAt         time.Time          `json:"verifiedAt"`
}
