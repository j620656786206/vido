package retry

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// MetadataSearcher defines the interface for metadata search operations
type MetadataSearcher interface {
	// SearchByTaskPayload performs a metadata search using the retry item's payload
	SearchByTaskPayload(ctx context.Context, payload json.RawMessage) error
}

// RetryExecutor implements TaskExecutor for retry operations
type RetryExecutor struct {
	metadataSearcher MetadataSearcher
	logger           *slog.Logger
}

// NewRetryExecutor creates a new RetryExecutor instance
func NewRetryExecutor(metadataSearcher MetadataSearcher, logger *slog.Logger) *RetryExecutor {
	if logger == nil {
		logger = slog.Default()
	}
	return &RetryExecutor{
		metadataSearcher: metadataSearcher,
		logger:           logger,
	}
}

// Execute implements TaskExecutor interface
// It executes the retry task based on its type
func (e *RetryExecutor) Execute(ctx context.Context, item *RetryItem) error {
	if item == nil {
		return fmt.Errorf("retry item is nil")
	}

	e.logger.Info("Executing retry task",
		"id", item.ID,
		"task_id", item.TaskID,
		"task_type", item.TaskType,
		"attempt", item.AttemptCount,
	)

	switch item.TaskType {
	case TaskTypeParse:
		return e.executeParse(ctx, item)
	case TaskTypeMetadataFetch:
		return e.executeMetadataFetch(ctx, item)
	default:
		return fmt.Errorf("unknown task type: %s", item.TaskType)
	}
}

// executeParse handles retry for parse tasks
func (e *RetryExecutor) executeParse(ctx context.Context, item *RetryItem) error {
	// Parse tasks typically involve re-parsing a filename and fetching metadata
	// The payload should contain the necessary information to retry
	if e.metadataSearcher == nil {
		return fmt.Errorf("metadata searcher not configured")
	}

	return e.metadataSearcher.SearchByTaskPayload(ctx, item.Payload)
}

// executeMetadataFetch handles retry for metadata fetch tasks
func (e *RetryExecutor) executeMetadataFetch(ctx context.Context, item *RetryItem) error {
	if e.metadataSearcher == nil {
		return fmt.Errorf("metadata searcher not configured")
	}

	return e.metadataSearcher.SearchByTaskPayload(ctx, item.Payload)
}

// Compile-time interface verification
var _ TaskExecutor = (*RetryExecutor)(nil)

// RetryPayload represents the payload structure for retry tasks
type RetryPayload struct {
	// For parse/metadata_fetch tasks
	MediaID   string `json:"mediaId,omitempty"`
	Filename  string `json:"filename,omitempty"`
	MediaType string `json:"mediaType,omitempty"` // "movie" or "series"
	Title     string `json:"title,omitempty"`
	Year      int    `json:"year,omitempty"`
	Season    int    `json:"season,omitempty"`
	Episode   int    `json:"episode,omitempty"`
}

// ParsePayload parses the JSON payload into RetryPayload
func ParsePayload(data json.RawMessage) (*RetryPayload, error) {
	var payload RetryPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}
	return &payload, nil
}
