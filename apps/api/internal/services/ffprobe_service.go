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
	"strings"
	"time"
)

// ErrFFprobeNotAvailable is returned when ffprobe binary is not installed
var ErrFFprobeNotAvailable = errors.New("ffprobe not available")

// MediaTechInfo holds technical info extracted from a video file
type MediaTechInfo struct {
	VideoCodec      string          `json:"video_codec,omitempty"`
	VideoResolution string          `json:"video_resolution,omitempty"`
	AudioCodec      string          `json:"audio_codec,omitempty"`
	AudioChannels   int             `json:"audio_channels,omitempty"`
	HDRFormat       string          `json:"hdr_format,omitempty"`
	SubtitleTracks  []SubtitleTrack `json:"subtitle_tracks,omitempty"`
}

// SubtitleTrack represents a subtitle track (embedded or external)
type SubtitleTrack struct {
	Language string `json:"language"`
	Format   string `json:"format"`
	External bool   `json:"external"`
}

// FFprobeService extracts technical metadata from video files using ffprobe
type FFprobeService struct {
	semaphore chan struct{}
	timeout   time.Duration
	available bool
	logger    *slog.Logger
}

// NewFFprobeService creates a new FFprobeService.
// It checks if ffprobe is available at startup via exec.LookPath.
func NewFFprobeService(maxConcurrent int, timeout time.Duration, logger *slog.Logger) *FFprobeService {
	if logger == nil {
		logger = slog.Default()
	}
	if maxConcurrent < 1 {
		maxConcurrent = 3
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	svc := &FFprobeService{
		semaphore: make(chan struct{}, maxConcurrent),
		timeout:   timeout,
		logger:    logger.With("service", "ffprobe"),
	}

	if _, err := exec.LookPath("ffprobe"); err != nil {
		svc.logger.Warn("ffprobe not found — technical info extraction disabled")
		svc.available = false
	} else {
		svc.available = true
		svc.logger.Info("ffprobe available", "max_concurrent", maxConcurrent, "timeout", timeout)
	}

	return svc
}

// IsAvailable returns whether ffprobe is installed and usable
func (s *FFprobeService) IsAvailable() bool {
	return s.available
}

// Probe extracts technical info from a video file using ffprobe.
// Returns ErrFFprobeNotAvailable if ffprobe is not installed.
// Respects concurrency limit via semaphore and per-call timeout.
func (s *FFprobeService) Probe(ctx context.Context, filePath string) (*MediaTechInfo, error) {
	if !s.available {
		return nil, ErrFFprobeNotAvailable
	}

	// Acquire semaphore slot
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Create timeout context
	probeCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Execute ffprobe
	//nolint:gosec // filePath comes from trusted filesystem scan
	cmd := exec.CommandContext(probeCtx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		if probeCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("ffprobe timeout after %s: %s", s.timeout, filePath)
		}
		return nil, fmt.Errorf("ffprobe exec: %w", err)
	}

	info, err := parseFfprobeJSON(output)
	if err != nil {
		return nil, fmt.Errorf("ffprobe parse: %w", err)
	}

	s.logger.Debug("ffprobe extracted",
		"file", filePath,
		"video", info.VideoCodec,
		"resolution", info.VideoResolution,
		"audio", info.AudioCodec,
		"hdr", info.HDRFormat,
	)

	return info, nil
}

// ─── FFprobe JSON parsing ──────────────────────────────────────────────────

// ffprobeOutput represents the top-level ffprobe JSON output
type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

// ffprobeStream represents a single stream in ffprobe output
type ffprobeStream struct {
	CodecType     string         `json:"codec_type"`
	CodecName     string         `json:"codec_name"`
	Width         int            `json:"width,omitempty"`
	Height        int            `json:"height,omitempty"`
	Channels      int            `json:"channels,omitempty"`
	ColorTransfer string         `json:"color_transfer,omitempty"`
	SideDataList  []ffprobeSideData `json:"side_data_list,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

// ffprobeSideData represents side_data entries (used for Dolby Vision detection)
type ffprobeSideData struct {
	SideDataType string `json:"side_data_type"`
}

// ffprobeFormat represents the format section of ffprobe output
type ffprobeFormat struct {
	Filename string `json:"filename"`
	Size     string `json:"size"`
}

// parseFfprobeJSON parses ffprobe JSON output into MediaTechInfo
func parseFfprobeJSON(output []byte) (*MediaTechInfo, error) {
	var data ffprobeOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	info := &MediaTechInfo{}

	for _, stream := range data.Streams {
		switch stream.CodecType {
		case "video":
			if info.VideoCodec == "" { // Take first video stream
				info.VideoCodec = normalizeVideoCodec(stream.CodecName)
				if stream.Width > 0 && stream.Height > 0 {
					info.VideoResolution = fmt.Sprintf("%dx%d", stream.Width, stream.Height)
				}
				info.HDRFormat = detectHDR(stream)
			}
		case "audio":
			if info.AudioCodec == "" { // Take first audio stream
				info.AudioCodec = normalizeAudioCodec(stream.CodecName)
				info.AudioChannels = stream.Channels
			}
		case "subtitle":
			lang := stream.Tags["language"]
			if lang == "" {
				lang = "und"
			}
			info.SubtitleTracks = append(info.SubtitleTracks, SubtitleTrack{
				Language: lang,
				Format:   stream.CodecName,
				External: false,
			})
		}
	}

	return info, nil
}

// ─── External subtitle detection ───────────────────────────────────────────

// subtitleExtensions maps sidecar subtitle file extensions to format names
var subtitleExtensions = map[string]string{
	".srt": "srt",
	".ass": "ass",
	".ssa": "ssa",
	".sub": "sub",
}

// DetectExternalSubtitles scans for sidecar subtitle files next to a video file.
// It looks for files like Movie.eng.srt, Movie.srt, Movie.zh-Hant.ass etc.
func DetectExternalSubtitles(videoPath string) []SubtitleTrack {
	if videoPath == "" {
		return nil
	}

	dir := filepath.Dir(videoPath)
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var tracks []SubtitleTrack
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		format, isSub := subtitleExtensions[ext]
		if !isSub {
			continue
		}

		// Check if subtitle belongs to this video (same base name)
		nameNoExt := strings.TrimSuffix(name, ext)
		if !strings.HasPrefix(nameNoExt, base) {
			continue
		}

		// Extract language tag: Movie.eng.srt → "eng", Movie.srt → "und"
		lang := "und"
		suffix := strings.TrimPrefix(nameNoExt, base)
		suffix = strings.TrimPrefix(suffix, ".")
		if suffix != "" {
			lang = suffix
		}

		tracks = append(tracks, SubtitleTrack{
			Language: lang,
			Format:   format,
			External: true,
		})
	}

	return tracks
}

// MergeSubtitleTracks combines embedded tracks (from FFprobe) with external sidecar tracks
func MergeSubtitleTracks(embedded, external []SubtitleTrack) []SubtitleTrack {
	result := make([]SubtitleTrack, 0, len(embedded)+len(external))
	result = append(result, embedded...)
	result = append(result, external...)
	return result
}

// ─── Codec normalization ───────────────────────────────────────────────────

var videoCodecMap = map[string]string{
	"hevc":    "H.265",
	"h265":    "H.265",
	"h264":    "H.264",
	"avc":     "H.264",
	"av1":     "AV1",
	"vp9":     "VP9",
	"mpeg4":   "MPEG-4",
	"mpeg2video": "MPEG-2",
}

var audioCodecMap = map[string]string{
	"dts":     "DTS",
	"dca":     "DTS",
	"aac":     "AAC",
	"ac3":     "AC-3",
	"eac3":    "E-AC-3",
	"truehd":  "TrueHD Atmos",
	"flac":    "FLAC",
	"pcm_s16le": "PCM",
	"pcm_s24le": "PCM",
	"opus":    "Opus",
	"vorbis":  "Vorbis",
	"mp3":     "MP3",
}

func normalizeVideoCodec(codec string) string {
	if normalized, ok := videoCodecMap[strings.ToLower(codec)]; ok {
		return normalized
	}
	return strings.ToUpper(codec)
}

func normalizeAudioCodec(codec string) string {
	if normalized, ok := audioCodecMap[strings.ToLower(codec)]; ok {
		return normalized
	}
	return strings.ToUpper(codec)
}

// ─── HDR detection ─────────────────────────────────────────────────────────

func detectHDR(stream ffprobeStream) string {
	// Check for Dolby Vision via side_data
	for _, sd := range stream.SideDataList {
		if strings.Contains(strings.ToLower(sd.SideDataType), "dolby vision") {
			return "Dolby Vision"
		}
	}

	// Check color_transfer for HDR10 / HLG
	switch strings.ToLower(stream.ColorTransfer) {
	case "smpte2084":
		return "HDR10"
	case "arib-std-b67":
		return "HLG"
	}

	return "" // SDR
}
