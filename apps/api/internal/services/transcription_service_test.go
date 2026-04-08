package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vido/api/internal/ai"
)

func TestTranscriptionService_IsAvailable_NoExtractor(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	assert.False(t, svc.IsAvailable())
}

func TestTranscriptionService_IsAvailable_ExtractorNotAvailable(t *testing.T) {
	extractor := &AudioExtractorService{available: false}
	svc := NewTranscriptionService(extractor, nil, nil, nil)
	assert.False(t, svc.IsAvailable())
}

func TestTranscriptionService_IsAvailable_NoWhisper(t *testing.T) {
	extractor := &AudioExtractorService{available: true}
	svc := NewTranscriptionService(extractor, nil, nil, nil)
	assert.False(t, svc.IsAvailable())
}

func TestTranscriptionService_IsInProgress_Empty(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	assert.False(t, svc.IsInProgress(1))
}

func TestTranscriptionService_IsInProgress_Set(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.mu.Lock()
	svc.inProgress[42] = "job-123"
	svc.mu.Unlock()

	assert.True(t, svc.IsInProgress(42))
	assert.False(t, svc.IsInProgress(99))
}

func TestTranscriptionService_StartTranscription_Disabled(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	_, err := svc.StartTranscription(nil, 1, "/test.mkv", "/media")
	assert.ErrorIs(t, err, ErrTranscriptionDisabled)
}

func TestTranscriptionService_StartTranscription_AlreadyInProgress(t *testing.T) {
	extractor := &AudioExtractorService{
		available: true,
		semaphore: make(chan struct{}, 1),
	}
	whisperClient := ai.NewWhisperClient("test-key")

	svc := NewTranscriptionService(extractor, whisperClient, nil, nil)

	// Manually set in-progress
	svc.mu.Lock()
	svc.inProgress[42] = "existing-job"
	svc.mu.Unlock()

	_, err := svc.StartTranscription(nil, 42, "/test.mkv", "/media")
	assert.ErrorIs(t, err, ErrTranscriptionInProgress)
}

func TestTranscriptionEventTypes(t *testing.T) {
	// Verify event type constants match expected SSE event names
	assert.Equal(t, "transcription_extracting", string(EventTranscriptionExtracting))
	assert.Equal(t, "transcription_progress", string(EventTranscriptionProgress))
	assert.Equal(t, "transcription_complete", string(EventTranscriptionComplete))
	assert.Equal(t, "transcription_failed", string(EventTranscriptionFailed))
}
