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

// ExtractSlot is the narrow port over the process-wide extraction gate
// (subtitle.ExtractGate). Audio extraction for speech recognition is ffmpeg
// doing a FULL decode of the same file on the same disk as subtitle
// extraction, so the two must take turns or they fight over one spindle
// exactly the way two subtitle extractions did (eval-1 finding 7, sub-6-3
// CR M3). services ↛ subtitle (Rule 19), so the concrete gate is injected
// from main.go through this one method.
type ExtractSlot interface {
	// Acquire blocks until the slot is free and returns the release func.
	Acquire(ctx context.Context) (release func(), err error)
}

// AudioExtractorService extracts audio tracks from video files using FFmpeg.
// Follows FFprobeService pattern: semaphore for concurrency, timeout, graceful degradation.
type AudioExtractorService struct {
	semaphore chan struct{}
	// timeout is the configured FLOOR of one extraction; the effective bound
	// grows with file size (SizedFFmpegTimeout) — a 66.8 GB remux needed 4:55
	// of the old fixed 5 minutes on the NAS.
	timeout      time.Duration
	perGBTimeout time.Duration
	// fileSize answers "how big is this file" for the size-aware bound; a
	// func, not an interface (Rule 11). Tests override it.
	fileSize func(path string) (int64, error)
	// slot is the shared disk gate (optional). nil = this service serializes
	// only against itself, the pre-sub-6-3 behaviour.
	slot      ExtractSlot
	available bool
	logger    *slog.Logger
}

// AudioExtractorOption configures one optional dependency.
type AudioExtractorOption func(*AudioExtractorService)

// WithAudioExtractSlot shares the process-wide extraction gate.
func WithAudioExtractSlot(slot ExtractSlot) AudioExtractorOption {
	return func(s *AudioExtractorService) {
		if slot != nil {
			s.slot = slot
		}
	}
}

// WithAudioExtractPerGB overrides the size-aware allowance per gigabyte
// (SUBTITLE_EXTRACT_PER_GB_SECONDS — the same knob as subtitle extraction,
// same disk, same kind of full-file ffmpeg read). Non-positive values are
// ignored.
func WithAudioExtractPerGB(d time.Duration) AudioExtractorOption {
	return func(s *AudioExtractorService) {
		if d > 0 {
			s.perGBTimeout = d
		}
	}
}

// withAudioFileSize overrides the size lookup (tests).
func withAudioFileSize(fn func(path string) (int64, error)) AudioExtractorOption {
	return func(s *AudioExtractorService) {
		if fn != nil {
			s.fileSize = fn
		}
	}
}

// defaultAudioExtractPerGB mirrors the subtitle extractor's default.
const defaultAudioExtractPerGB = 30 * time.Second

func statAudioFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// NewAudioExtractorService creates a new AudioExtractorService.
// Checks if ffmpeg is available at startup via exec.LookPath (AC #4).
func NewAudioExtractorService(maxConcurrent int, timeout time.Duration, logger *slog.Logger, opts ...AudioExtractorOption) *AudioExtractorService {
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
		semaphore:    make(chan struct{}, maxConcurrent),
		timeout:      timeout,
		perGBTimeout: defaultAudioExtractPerGB,
		fileSize:     statAudioFileSize,
		logger:       logger.With("service", "audio_extractor"),
	}
	for _, opt := range opts {
		opt(svc)
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		svc.logger.Warn("ffmpeg not found — audio extraction disabled")
		svc.available = false
	} else {
		svc.available = true
		svc.logger.Info("ffmpeg available", "max_concurrent", maxConcurrent, "timeout", timeout, "per_gb_timeout", svc.perGBTimeout)
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

	// The shared disk gate goes around the ffmpeg pass ONLY, and outside the
	// timeout below: queueing behind another extraction must not eat the
	// budget this decode needs (sub-6-3).
	if s.slot != nil {
		release, err := s.slot.Acquire(ctx)
		if err != nil {
			return "", fmt.Errorf("audio extraction gave up waiting for the extraction slot: %w", err)
		}
		defer release()
	}

	timeout, sizeGB, knob := s.effectiveTimeout(inputPath)
	extractCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := time.Now()

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
		if errors.Is(extractCtx.Err(), context.DeadlineExceeded) {
			// The deadline can also be the CALLER's (a batch item's bound):
			// name our knob only when OUR bound fired, so the message never
			// blames a setting that did not apply. One line — it reaches the
			// screen through the transcription_failed event.
			if ctx.Err() == nil {
				return "", fmt.Errorf("%w: ffmpeg timed out after %s on %s (file %.1f GB, timeout %d s — raise %s for slow disks): %w",
					ErrAudioExtractionTimeout, time.Since(started).Round(time.Second), filepath.Base(inputPath),
					sizeGB, int(timeout.Seconds()), knob, context.DeadlineExceeded)
			}
			return "", fmt.Errorf("%w: ffmpeg stopped by the caller's deadline after %s on %s (file %.1f GB): %w",
				ErrAudioExtractionTimeout, time.Since(started).Round(time.Second), filepath.Base(inputPath),
				sizeGB, context.DeadlineExceeded)
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

// EffectiveTimeout is the deadline one ffmpeg pass over inputPath gets:
// max(configured floor, size × per-GB allowance) — the subtitle extractor's
// rule (sub-6-3 AC #1), now shared by the ASR audio extraction. A file whose
// size cannot be read gets the floor. The size in GB rides along for log
// lines; 0 when unknown.
func (s *AudioExtractorService) EffectiveTimeout(inputPath string) (time.Duration, float64) {
	timeout, gb, _ := s.effectiveTimeout(inputPath)
	return timeout, gb
}

func (s *AudioExtractorService) effectiveTimeout(inputPath string) (timeout time.Duration, sizeGB float64, knob string) {
	size, err := s.fileSize(inputPath)
	if err != nil || size <= 0 {
		timeout, knob = sizedFFmpegTimeoutKnob(s.timeout, s.perGBTimeout, 0)
		return timeout, 0, knob
	}
	timeout, knob = sizedFFmpegTimeoutKnob(s.timeout, s.perGBTimeout, size)
	return timeout, float64(size) / BytesPerGB, knob
}
