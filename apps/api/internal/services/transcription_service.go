package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/sse"
)

// SSE event types for transcription progress (AC #6)
const (
	EventTranscriptionExtracting sse.EventType = "transcription_extracting"
	EventTranscriptionProgress   sse.EventType = "transcription_progress"
	EventTranscriptionComplete   sse.EventType = "transcription_complete"
	EventTranscriptionFailed     sse.EventType = "transcription_failed"
)

// Transcription errors
var (
	ErrTranscriptionInProgress = errors.New("transcription already in progress for this media")
	ErrTranscriptionDisabled   = errors.New("transcription disabled: OpenAI API key not configured")
)

// TranscriptionResult holds the result of a transcription job.
type TranscriptionResult struct {
	JobID    string `json:"job_id"`
	MediaID  int64  `json:"media_id"`
	SRTPath  string `json:"srt_path"`
	Duration string `json:"duration"`
	Error    string `json:"error,omitempty"`
}

// TranscriptionService orchestrates the audio extraction → Whisper transcription pipeline.
type TranscriptionService struct {
	audioExtractor *AudioExtractorService
	whisperClient  *ai.WhisperClient
	sseHub         *sse.Hub
	logger         *slog.Logger
	timeout        time.Duration

	mu         sync.Mutex
	inProgress map[int64]string // mediaID → jobID
}

// NewTranscriptionService creates a new TranscriptionService.
func NewTranscriptionService(
	audioExtractor *AudioExtractorService,
	whisperClient *ai.WhisperClient,
	sseHub *sse.Hub,
	logger *slog.Logger,
) *TranscriptionService {
	if logger == nil {
		logger = slog.Default()
	}
	return &TranscriptionService{
		audioExtractor: audioExtractor,
		whisperClient:  whisperClient,
		sseHub:         sseHub,
		logger:         logger.With("service", "transcription"),
		timeout:        5 * time.Minute,
		inProgress:     make(map[int64]string),
	}
}

// IsAvailable returns true if both FFmpeg and Whisper API are configured.
func (s *TranscriptionService) IsAvailable() bool {
	return s.audioExtractor != nil && s.audioExtractor.IsAvailable() && s.whisperClient != nil
}

// IsInProgress returns true if a transcription is already running for the given media ID.
func (s *TranscriptionService) IsInProgress(mediaID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.inProgress[mediaID]
	return ok
}

// StartTranscription initiates an async transcription job for a media file.
// Returns a job ID immediately. Progress is reported via SSE events.
func (s *TranscriptionService) StartTranscription(ctx context.Context, mediaID int64, filePath string, mediaDir string) (string, error) {
	if !s.IsAvailable() {
		return "", ErrTranscriptionDisabled
	}

	s.mu.Lock()
	if _, exists := s.inProgress[mediaID]; exists {
		s.mu.Unlock()
		return "", ErrTranscriptionInProgress
	}
	jobID := uuid.New().String()
	s.inProgress[mediaID] = jobID
	s.mu.Unlock()

	// Run transcription pipeline in background goroutine with timeout
	go func() {
		pipelineCtx, pipelineCancel := context.WithTimeout(context.Background(), s.timeout)
		defer pipelineCancel()
		s.runPipeline(pipelineCtx, jobID, mediaID, filePath, mediaDir)
	}()

	return jobID, nil
}

// runPipeline executes the full transcription pipeline:
// Extract audio → (optional chunk) → Whisper API → Merge SRT → Save
func (s *TranscriptionService) runPipeline(ctx context.Context, jobID string, mediaID int64, filePath string, mediaDir string) {
	defer func() {
		s.mu.Lock()
		delete(s.inProgress, mediaID)
		s.mu.Unlock()
	}()

	startedAt := time.Now()

	// Phase 1: Extract audio
	s.broadcastEvent(EventTranscriptionExtracting, map[string]interface{}{
		"job_id":   jobID,
		"media_id": mediaID,
		"phase":    "extracting",
		"message":  "Extracting audio track from media file",
	})

	// List audio tracks and select English track
	tracks, err := s.audioExtractor.ListAudioTracks(ctx, filePath)
	if err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("list audio tracks: %v", err))
		return
	}

	selectedTrack, err := SelectEnglishTrack(tracks)
	if err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("select audio track: %v", err))
		return
	}

	s.logger.Info("audio track selected",
		"job_id", jobID,
		"track_index", selectedTrack.Index,
		"language", selectedTrack.Language,
	)

	// Extract audio to temp WAV
	audioPath, err := s.audioExtractor.ExtractAudio(ctx, filePath, selectedTrack.Index)
	if err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("extract audio: %v", err))
		return
	}
	defer os.Remove(audioPath)

	// Phase 2: Transcribe
	s.broadcastEvent(EventTranscriptionProgress, map[string]interface{}{
		"job_id":   jobID,
		"media_id": mediaID,
		"phase":    "transcribing",
		"message":  "Transcribing audio with Whisper API",
	})

	srtContent, err := s.transcribeAudio(ctx, audioPath)
	if err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("transcribe: %v", err))
		return
	}

	// Phase 3: Save SRT
	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	srtPath := filepath.Join(mediaDir, baseName+".en.srt")

	if err := os.WriteFile(srtPath, []byte(srtContent), 0644); err != nil {
		s.failJob(jobID, mediaID, fmt.Sprintf("save SRT: %v", err))
		return
	}

	duration := time.Since(startedAt).Round(time.Second).String()

	s.logger.Info("transcription complete",
		"job_id", jobID,
		"media_id", mediaID,
		"srt_path", srtPath,
		"duration", duration,
	)

	// Phase 4: Complete
	s.broadcastEvent(EventTranscriptionComplete, map[string]interface{}{
		"job_id":   jobID,
		"media_id": mediaID,
		"phase":    "complete",
		"srt_path": srtPath,
		"duration": duration,
		"message":  "Transcription complete",
	})
}

// transcribeAudio handles chunking and multi-part transcription for large files (AC #7).
func (s *TranscriptionService) transcribeAudio(ctx context.Context, audioPath string) (string, error) {
	needsChunk, err := ai.NeedsChunking(audioPath)
	if err != nil {
		return "", fmt.Errorf("check chunking: %w", err)
	}

	if !needsChunk {
		return s.whisperClient.Transcribe(ctx, audioPath)
	}

	// Split and transcribe chunks
	s.logger.Info("audio exceeds 25MB, splitting into chunks")
	chunks, err := ai.SplitAudioChunks(ctx, audioPath)
	if err != nil {
		return "", fmt.Errorf("split chunks: %w", err)
	}

	// Cleanup chunk files (skip first if it's the original)
	defer func() {
		for _, chunk := range chunks {
			if chunk != audioPath {
				os.Remove(chunk)
			}
		}
	}()

	var srtChunks []string
	for i, chunkPath := range chunks {
		s.logger.Info("transcribing chunk",
			"chunk", i+1,
			"total", len(chunks),
		)
		srt, err := s.whisperClient.Transcribe(ctx, chunkPath)
		if err != nil {
			return "", fmt.Errorf("transcribe chunk %d/%d: %w", i+1, len(chunks), err)
		}
		srtChunks = append(srtChunks, srt)
	}

	return ai.MergeSRTChunks(srtChunks, ai.WhisperChunkDuration), nil
}

func (s *TranscriptionService) failJob(jobID string, mediaID int64, errMsg string) {
	s.logger.Error("transcription failed",
		"job_id", jobID,
		"media_id", mediaID,
		"error", errMsg,
	)
	s.broadcastEvent(EventTranscriptionFailed, map[string]interface{}{
		"job_id":   jobID,
		"media_id": mediaID,
		"phase":    "failed",
		"error":    errMsg,
		"message":  "Transcription failed: " + errMsg,
	})
}

func (s *TranscriptionService) broadcastEvent(eventType sse.EventType, data interface{}) {
	if s.sseHub == nil {
		return
	}
	s.sseHub.Broadcast(sse.Event{
		ID:   uuid.New().String(),
		Type: eventType,
		Data: data,
	})
}
