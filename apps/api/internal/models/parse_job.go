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
	TorrentHash  string         `json:"torrentHash"`
	FilePath     string         `json:"filePath"`
	FileName     string         `json:"fileName"`
	Status       ParseJobStatus `json:"status"`
	MediaID      *string        `json:"mediaId,omitempty"`
	ErrorMessage *string        `json:"errorMessage,omitempty"`
	RetryCount   int            `json:"retryCount"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	CompletedAt  *time.Time     `json:"completedAt,omitempty"`
}
