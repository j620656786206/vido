package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Audio extraction errors
var (
	ErrFFmpegNotAvailable     = errors.New("ffmpeg not available")
	ErrNoAudioTrack           = errors.New("no audio track found in media file")
	ErrAudioExtractionFailed  = errors.New("audio extraction failed")
	ErrAudioExtractionTimeout = errors.New("audio extraction timed out")
)

// AudioTrack represents an audio stream in a media file.
type AudioTrack struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Codec    string `json:"codec"`
	Channels int    `json:"channels"`
}

// AudioExtractorService extracts audio tracks from video files using FFmpeg.
// Follows FFprobeService pattern: semaphore for concurrency, timeout, graceful degradation.
type AudioExtractorService struct {
	semaphore chan struct{}
	timeout   time.Duration
	available bool
	logger    *slog.Logger
}

// NewAudioExtractorService creates a new AudioExtractorService.
// Checks if ffmpeg is available at startup via exec.LookPath (AC #4).
func NewAudioExtractorService(maxConcurrent int, timeout time.Duration, logger *slog.Logger) *AudioExtractorService {
	if logger == nil {
		logger = slog.Default()
	}
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	svc := &AudioExtractorService{
		semaphore: make(chan struct{}, maxConcurrent),
		timeout:   timeout,
		logger:    logger.With("service", "audio_extractor"),
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		svc.logger.Warn("ffmpeg not found — audio extraction disabled")
		svc.available = false
	} else {
		svc.available = true
		svc.logger.Info("ffmpeg available", "max_concurrent", maxConcurrent, "timeout", timeout)
	}

	return svc
}

// IsAvailable returns whether ffmpeg is installed and usable.
func (s *AudioExtractorService) IsAvailable() bool {
	return s.available
}

// ListAudioTracks returns the audio tracks in a media file using ffprobe.
func (s *AudioExtractorService) ListAudioTracks(ctx context.Context, filePath string) ([]AudioTrack, error) {
	if !s.available {
		return nil, ErrFFmpegNotAvailable
	}

	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	//nolint:gosec // filePath comes from trusted DB record
	cmd := exec.CommandContext(probeCtx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-select_streams", "a",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		if probeCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("ffprobe timeout: %s", filePath)
		}
		return nil, fmt.Errorf("ffprobe exec: %w", err)
	}

	return parseAudioStreams(output)
}

// SelectEnglishTrack selects the English audio track from a list, falling back to the first track (AC #2).
func SelectEnglishTrack(tracks []AudioTrack) (AudioTrack, error) {
	if len(tracks) == 0 {
		return AudioTrack{}, ErrNoAudioTrack
	}

	// Prefer English track by language tag
	for _, t := range tracks {
		if t.Language == "eng" || t.Language == "en" {
			return t, nil
		}
	}

	// Fall back to first audio stream
	return tracks[0], nil
}

// ExtractAudio extracts an audio track from a media file to a WAV file (16kHz mono PCM).
// The output is written to OS temp dir. Caller is responsible for cleanup.
// Returns the path to the extracted WAV file.
func (s *AudioExtractorService) ExtractAudio(ctx context.Context, inputPath string, trackIndex int) (string, error) {
	if !s.available {
		return "", ErrFFmpegNotAvailable
	}

	// Acquire semaphore slot
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	extractCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Create temp file for output
	tmpFile, err := os.CreateTemp("", "vido-audio-*.wav")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	outputPath := tmpFile.Name()
	tmpFile.Close()

	// Extract audio: select specific stream, convert to 16kHz mono PCM WAV
	//nolint:gosec // inputPath comes from trusted DB record
	cmd := exec.CommandContext(extractCtx, "ffmpeg",
		"-i", inputPath,
		"-map", fmt.Sprintf("0:%d", trackIndex),
		"-vn",
		"-acodec", "pcm_s16le",
		"-ar", "16000",
		"-ac", "1",
		"-y",
		outputPath,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		os.Remove(outputPath)
		if extractCtx.Err() == context.DeadlineExceeded {
			return "", ErrAudioExtractionTimeout
		}
		s.logger.Error("ffmpeg extraction failed",
			"error", err,
			"input", filepath.Base(inputPath),
			"output", string(output),
		)
		return "", fmt.Errorf("%w: %v", ErrAudioExtractionFailed, err)
	}

	s.logger.Info("audio extracted",
		"input", filepath.Base(inputPath),
		"track_index", trackIndex,
		"output", outputPath,
	)

	return outputPath, nil
}

// ─── ffprobe audio stream parsing ─────────────────────────────────────────

type ffprobeAudioOutput struct {
	Streams []ffprobeAudioStream `json:"streams"`
}

type ffprobeAudioStream struct {
	Index     int               `json:"index"`
	CodecName string            `json:"codec_name"`
	Channels  int               `json:"channels"`
	Tags      map[string]string `json:"tags,omitempty"`
}

func parseAudioStreams(output []byte) ([]AudioTrack, error) {
	var data ffprobeAudioOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("unmarshal audio streams: %w", err)
	}

	tracks := make([]AudioTrack, 0, len(data.Streams))
	for _, s := range data.Streams {
		lang := s.Tags["language"]
		if lang == "" {
			lang = "und"
		}
		tracks = append(tracks, AudioTrack{
			Index:    s.Index,
			Language: lang,
			Codec:    s.CodecName,
			Channels: s.Channels,
		})
	}

	return tracks, nil
}
