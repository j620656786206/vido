package models

import "time"

// ParseJobStatus represents the status of a parse job in the queue.
type ParseJobStatus string

const (
	ParseJobPending    ParseJobStatus = "pending"
	ParseJobProcessing ParseJobStatus = "processing"
	ParseJobCompleted  ParseJobStatus = "completed"
	ParseJobFailed     ParseJobStatus = "failed"
	ParseJobSkipped    ParseJobStatus = "skipped" // Duplicate or already in library
)

// ParseJob represents a queued job to parse a completed download.
type ParseJob struct {
	ID           string         `json:"id"`
	TorrentHash  string         `json:"torrent_hash"`
	FilePath     string         `json:"file_path"`
	FileName     string         `json:"file_name"`
	Status       ParseJobStatus `json:"status"`
	MediaID      *string        `json:"media_id,omitempty"`
	ErrorMessage *string        `json:"error_message,omitempty"`
	RetryCount   int            `json:"retry_count"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
}
