package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJobStatus_Values(t *testing.T) {
	assert.Equal(t, ParseJobStatus("pending"), ParseJobPending)
	assert.Equal(t, ParseJobStatus("processing"), ParseJobProcessing)
	assert.Equal(t, ParseJobStatus("completed"), ParseJobCompleted)
	assert.Equal(t, ParseJobStatus("failed"), ParseJobFailed)
	assert.Equal(t, ParseJobStatus("skipped"), ParseJobSkipped)
}

func TestParseJob_JSONSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	mediaID := "media-123"
	errMsg := "parse failed"

	job := &ParseJob{
		ID:           "job-1",
		TorrentHash:  "abc123",
		FilePath:     "/downloads/movie.mkv",
		FileName:     "movie.mkv",
		Status:       ParseJobPending,
		MediaID:      &mediaID,
		ErrorMessage: &errMsg,
		RetryCount:   2,
		CreatedAt:    now,
		UpdatedAt:    now,
		CompletedAt:  &now,
	}

	data, err := json.Marshal(job)
	require.NoError(t, err)

	var decoded ParseJob
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, job.ID, decoded.ID)
	assert.Equal(t, job.TorrentHash, decoded.TorrentHash)
	assert.Equal(t, job.FilePath, decoded.FilePath)
	assert.Equal(t, job.FileName, decoded.FileName)
	assert.Equal(t, job.Status, decoded.Status)
	assert.Equal(t, *job.MediaID, *decoded.MediaID)
	assert.Equal(t, *job.ErrorMessage, *decoded.ErrorMessage)
	assert.Equal(t, job.RetryCount, decoded.RetryCount)
}

func TestParseJob_JSONOmitsOptionalNilFields(t *testing.T) {
	job := &ParseJob{
		ID:          "job-2",
		TorrentHash: "def456",
		FilePath:    "/downloads/show.mkv",
		FileName:    "show.mkv",
		Status:      ParseJobPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	data, err := json.Marshal(job)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	_, hasMediaID := raw["mediaId"]
	_, hasError := raw["errorMessage"]
	_, hasCompleted := raw["completedAt"]

	assert.False(t, hasMediaID, "mediaId should be omitted when nil")
	assert.False(t, hasError, "errorMessage should be omitted when nil")
	assert.False(t, hasCompleted, "completedAt should be omitted when nil")
}
