package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandardParseSteps(t *testing.T) {
	steps := StandardParseSteps()

	assert.Len(t, steps, 6)

	expectedSteps := []struct {
		name  string
		label string
	}{
		{"filename_extract", "解析檔名"},
		{"tmdb_search", "搜尋 TMDb"},
		{"douban_search", "搜尋豆瓣"},
		{"wikipedia_search", "搜尋 Wikipedia"},
		{"ai_retry", "AI 重試"},
		{"download_poster", "下載海報"},
	}

	for i, expected := range expectedSteps {
		assert.Equal(t, expected.name, steps[i].Name)
		assert.Equal(t, expected.label, steps[i].Label)
		assert.Equal(t, StepPending, steps[i].Status)
	}
}

func TestNewParseProgress(t *testing.T) {
	progress := NewParseProgress("task-123", "test-movie.mkv")

	assert.Equal(t, "task-123", progress.TaskID)
	assert.Equal(t, "test-movie.mkv", progress.Filename)
	assert.Equal(t, ParseStatusPending, progress.Status)
	assert.Len(t, progress.Steps, 6)
	assert.Equal(t, 0, progress.CurrentStep)
	assert.Equal(t, 0, progress.Percentage)
	assert.NotZero(t, progress.StartedAt)
	assert.Nil(t, progress.CompletedAt)
	assert.Nil(t, progress.Result)
}

func TestParseProgress_StartStep(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	progress.StartStep(0)

	assert.Equal(t, StepInProgress, progress.Steps[0].Status)
	assert.NotNil(t, progress.Steps[0].StartedAt)
	assert.Equal(t, 0, progress.CurrentStep)
}

func TestParseProgress_StartStep_InvalidIndex(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	// Should not panic on invalid index
	progress.StartStep(-1)
	progress.StartStep(100)

	// All steps should still be pending
	for _, step := range progress.Steps {
		assert.Equal(t, StepPending, step.Status)
	}
}

func TestParseProgress_CompleteStep(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	progress.StartStep(0)
	progress.CompleteStep(0)

	assert.Equal(t, StepSuccess, progress.Steps[0].Status)
	assert.NotNil(t, progress.Steps[0].EndedAt)
	// 1 out of 6 steps = 16%
	assert.Equal(t, 16, progress.Percentage)
}

func TestParseProgress_FailStep(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	progress.StartStep(1)
	progress.FailStep(1, "TMDb API timeout")

	assert.Equal(t, StepFailed, progress.Steps[1].Status)
	assert.NotNil(t, progress.Steps[1].EndedAt)
	assert.Equal(t, "TMDb API timeout", progress.Steps[1].Error)
}

func TestParseProgress_SkipStep(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	progress.SkipStep(2)

	assert.Equal(t, StepSkipped, progress.Steps[2].Status)
	assert.NotNil(t, progress.Steps[2].EndedAt)
	// Skipped steps count toward completion percentage
	assert.Equal(t, 16, progress.Percentage)
}

func TestParseProgress_Complete(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	result := &ParseResult{
		MediaID:        "movie-456",
		Title:          "Test Movie",
		Year:           2024,
		MediaType:      "movie",
		MetadataSource: MetadataSourceTMDb,
		Confidence:     0.95,
	}

	progress.Complete(result)

	assert.Equal(t, ParseStatusSuccess, progress.Status)
	assert.NotNil(t, progress.CompletedAt)
	assert.Equal(t, 100, progress.Percentage)
	require.NotNil(t, progress.Result)
	assert.Equal(t, "movie-456", progress.Result.MediaID)
	assert.Equal(t, "Test Movie", progress.Result.Title)
	assert.Equal(t, 2024, progress.Result.Year)
}

func TestParseProgress_CompleteWithWarning(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	progress.CompleteWithWarning("Manual selection required")

	assert.Equal(t, ParseStatusNeedsAI, progress.Status)
	assert.NotNil(t, progress.CompletedAt)
	assert.Equal(t, "Manual selection required", progress.Message)
}

func TestParseProgress_Fail(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	progress.Fail("All sources failed")

	assert.Equal(t, ParseStatusFailed, progress.Status)
	assert.NotNil(t, progress.CompletedAt)
	assert.Equal(t, "All sources failed", progress.Message)
}

func TestParseProgress_UpdatePercentage(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	// Complete 3 steps (success or skipped)
	progress.CompleteStep(0) // 16%
	progress.CompleteStep(1) // 33%
	progress.SkipStep(2)     // 50%

	assert.Equal(t, 50, progress.Percentage)
}

func TestParseProgress_GetStepByName(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	step := progress.GetStepByName("tmdb_search")

	require.NotNil(t, step)
	assert.Equal(t, "tmdb_search", step.Name)
	assert.Equal(t, "搜尋 TMDb", step.Label)
}

func TestParseProgress_GetStepByName_NotFound(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	step := progress.GetStepByName("nonexistent")

	assert.Nil(t, step)
}

func TestParseProgress_GetStepIndex(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	assert.Equal(t, 0, progress.GetStepIndex("filename_extract"))
	assert.Equal(t, 1, progress.GetStepIndex("tmdb_search"))
	assert.Equal(t, 5, progress.GetStepIndex("download_poster"))
	assert.Equal(t, -1, progress.GetStepIndex("nonexistent"))
}

func TestParseProgress_HasFailedSteps(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	assert.False(t, progress.HasFailedSteps())

	progress.FailStep(1, "Error")
	assert.True(t, progress.HasFailedSteps())
}

func TestParseProgress_GetFailedSteps(t *testing.T) {
	progress := NewParseProgress("task-123", "test.mkv")

	progress.CompleteStep(0)
	progress.FailStep(1, "TMDb failed")
	progress.FailStep(2, "Douban failed")
	progress.CompleteStep(3)

	failed := progress.GetFailedSteps()

	assert.Len(t, failed, 2)
	assert.Equal(t, "tmdb_search", failed[0].Name)
	assert.Equal(t, "TMDb failed", failed[0].Error)
	assert.Equal(t, "douban_search", failed[1].Name)
	assert.Equal(t, "Douban failed", failed[1].Error)
}

func TestParseProgress_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		status   ParseStatus
		expected bool
	}{
		{"pending", ParseStatusPending, false},
		{"parsing", ParseStatusParsing, false},
		{"success", ParseStatusSuccess, true},
		{"failed", ParseStatusFailed, true},
		{"needs_ai", ParseStatusNeedsAI, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress := NewParseProgress("task-123", "test.mkv")
			progress.Status = tt.status

			assert.Equal(t, tt.expected, progress.IsComplete())
		})
	}
}

func TestStepStatus_Values(t *testing.T) {
	assert.Equal(t, StepStatus("pending"), StepPending)
	assert.Equal(t, StepStatus("in_progress"), StepInProgress)
	assert.Equal(t, StepStatus("success"), StepSuccess)
	assert.Equal(t, StepStatus("failed"), StepFailed)
	assert.Equal(t, StepStatus("skipped"), StepSkipped)
}

func TestParseResult_Fields(t *testing.T) {
	result := ParseResult{
		MediaID:        "movie-123",
		Title:          "Test Movie",
		Year:           2024,
		MediaType:      "movie",
		MetadataSource: MetadataSourceTMDb,
		Confidence:     0.95,
	}

	assert.Equal(t, "movie-123", result.MediaID)
	assert.Equal(t, "Test Movie", result.Title)
	assert.Equal(t, 2024, result.Year)
	assert.Equal(t, "movie", result.MediaType)
	assert.Equal(t, MetadataSourceTMDb, result.MetadataSource)
	assert.Equal(t, 0.95, result.Confidence)
}
