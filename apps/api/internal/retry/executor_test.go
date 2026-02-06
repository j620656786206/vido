package retry

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMetadataSearcher implements MetadataSearcher for testing
type MockMetadataSearcher struct {
	shouldFail bool
	lastPayload json.RawMessage
}

func (m *MockMetadataSearcher) SearchByTaskPayload(ctx context.Context, payload json.RawMessage) error {
	m.lastPayload = payload
	if m.shouldFail {
		return errors.New("search failed")
	}
	return nil
}

func TestNewRetryExecutor(t *testing.T) {
	searcher := &MockMetadataSearcher{}
	executor := NewRetryExecutor(searcher, nil)

	assert.NotNil(t, executor)
	assert.NotNil(t, executor.logger)
	assert.Equal(t, searcher, executor.metadataSearcher)
}

func TestRetryExecutor_Execute_NilItem(t *testing.T) {
	executor := NewRetryExecutor(nil, nil)

	err := executor.Execute(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestRetryExecutor_Execute_Parse(t *testing.T) {
	searcher := &MockMetadataSearcher{}
	executor := NewRetryExecutor(searcher, nil)

	payload, _ := json.Marshal(RetryPayload{
		Title:     "Test Movie",
		MediaType: "movie",
		Year:      2024,
	})

	item := &RetryItem{
		ID:       "test-1",
		TaskID:   "task-1",
		TaskType: TaskTypeParse,
		Payload:  payload,
	}

	err := executor.Execute(context.Background(), item)
	assert.NoError(t, err)
	assert.NotNil(t, searcher.lastPayload)
}

func TestRetryExecutor_Execute_MetadataFetch(t *testing.T) {
	searcher := &MockMetadataSearcher{}
	executor := NewRetryExecutor(searcher, nil)

	payload, _ := json.Marshal(RetryPayload{
		Title:     "Test Series",
		MediaType: "tv",
		Year:      2024,
	})

	item := &RetryItem{
		ID:       "test-1",
		TaskID:   "task-1",
		TaskType: TaskTypeMetadataFetch,
		Payload:  payload,
	}

	err := executor.Execute(context.Background(), item)
	assert.NoError(t, err)
}

func TestRetryExecutor_Execute_UnknownTaskType(t *testing.T) {
	searcher := &MockMetadataSearcher{}
	executor := NewRetryExecutor(searcher, nil)

	item := &RetryItem{
		ID:       "test-1",
		TaskID:   "task-1",
		TaskType: "unknown_type",
		Payload:  []byte("{}"),
	}

	err := executor.Execute(context.Background(), item)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown task type")
}

func TestRetryExecutor_Execute_NoSearcher(t *testing.T) {
	executor := NewRetryExecutor(nil, nil)

	item := &RetryItem{
		ID:       "test-1",
		TaskID:   "task-1",
		TaskType: TaskTypeParse,
		Payload:  []byte("{}"),
	}

	err := executor.Execute(context.Background(), item)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestRetryExecutor_Execute_SearchFails(t *testing.T) {
	searcher := &MockMetadataSearcher{shouldFail: true}
	executor := NewRetryExecutor(searcher, nil)

	payload, _ := json.Marshal(RetryPayload{
		Title:     "Test Movie",
		MediaType: "movie",
	})

	item := &RetryItem{
		ID:       "test-1",
		TaskID:   "task-1",
		TaskType: TaskTypeParse,
		Payload:  payload,
	}

	err := executor.Execute(context.Background(), item)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search failed")
}

func TestParsePayload(t *testing.T) {
	tests := []struct {
		name    string
		data    json.RawMessage
		wantErr bool
	}{
		{
			name:    "valid payload",
			data:    json.RawMessage(`{"title":"Test Movie","mediaType":"movie","year":2024}`),
			wantErr: false,
		},
		{
			name:    "empty payload",
			data:    json.RawMessage(`{}`),
			wantErr: false,
		},
		{
			name:    "invalid json",
			data:    json.RawMessage(`{invalid`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := ParsePayload(tt.data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, payload)
			}
		})
	}
}

func TestRetryPayload_Fields(t *testing.T) {
	data := json.RawMessage(`{
		"mediaId": "media-123",
		"filename": "test.mkv",
		"mediaType": "movie",
		"title": "Test Movie",
		"year": 2024,
		"season": 1,
		"episode": 5
	}`)

	payload, err := ParsePayload(data)
	require.NoError(t, err)

	assert.Equal(t, "media-123", payload.MediaID)
	assert.Equal(t, "test.mkv", payload.Filename)
	assert.Equal(t, "movie", payload.MediaType)
	assert.Equal(t, "Test Movie", payload.Title)
	assert.Equal(t, 2024, payload.Year)
	assert.Equal(t, 1, payload.Season)
	assert.Equal(t, 5, payload.Episode)
}

// Verify interface implementation
var _ TaskExecutor = (*RetryExecutor)(nil)
