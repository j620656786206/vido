package ai

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	// WhisperAPIURL is the OpenAI Whisper transcription endpoint.
	WhisperAPIURL = "https://api.openai.com/v1/audio/transcriptions"
	// WhisperModel is the model identifier for Whisper.
	WhisperModel = "whisper-1"
	// WhisperMaxFileSize is the maximum file size the Whisper API accepts (25MB).
	WhisperMaxFileSize = 25 * 1024 * 1024
	// WhisperChunkDuration is the duration of each audio chunk in seconds (10 minutes).
	WhisperChunkDuration = 600
)

// Whisper API errors
var (
	ErrWhisperNotConfigured = errors.New("whisper: OpenAI API key not configured")
	ErrWhisperAPIError      = errors.New("whisper: API error")
	ErrWhisperTimeout       = errors.New("whisper: request timed out")
)

// WhisperClient transcribes audio files using the OpenAI Whisper API.
type WhisperClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	logger     *slog.Logger
}

// WhisperOption is a functional option for configuring WhisperClient.
type WhisperOption func(*WhisperClient)

// WithWhisperBaseURL sets a custom base URL (useful for testing).
func WithWhisperBaseURL(url string) WhisperOption {
	return func(c *WhisperClient) {
		c.baseURL = url
	}
}

// WithWhisperHTTPClient sets a custom HTTP client.
func WithWhisperHTTPClient(client *http.Client) WhisperOption {
	return func(c *WhisperClient) {
		c.httpClient = client
	}
}

// WithWhisperTimeout sets a custom timeout per request.
func WithWhisperTimeout(timeout time.Duration) WhisperOption {
	return func(c *WhisperClient) {
		c.timeout = timeout
	}
}

// NewWhisperClient creates a new Whisper API client.
func NewWhisperClient(apiKey string, opts ...WhisperOption) *WhisperClient {
	c := &WhisperClient{
		apiKey:  apiKey,
		baseURL: WhisperAPIURL,
		timeout: 5 * time.Minute,
		logger:  slog.Default().With("service", "whisper"),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Timeout: c.timeout,
		}
	}

	return c
}

// Transcribe sends an audio file to the Whisper API and returns the SRT transcription.
func (c *WhisperClient) Transcribe(ctx context.Context, audioPath string) (string, error) {
	if c.apiKey == "" {
		return "", ErrWhisperNotConfigured
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Build multipart form body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add audio file
	file, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("whisper: open audio file: %w", err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", fmt.Errorf("whisper: create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("whisper: copy audio data: %w", err)
	}

	// Add model and response format fields
	if err := writer.WriteField("model", WhisperModel); err != nil {
		return "", fmt.Errorf("whisper: write model field: %w", err)
	}
	if err := writer.WriteField("response_format", "srt"); err != nil {
		return "", fmt.Errorf("whisper: write format field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("whisper: close writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, body)
	if err != nil {
		return "", fmt.Errorf("whisper: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	c.logger.Debug("Whisper API request", "file", filepath.Base(audioPath))

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", ErrWhisperTimeout
		}
		return "", fmt.Errorf("%w: %v", ErrWhisperAPIError, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("whisper: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("Whisper API error",
			"status_code", resp.StatusCode,
			"body", string(respBody),
		)
		return "", fmt.Errorf("%w: status %d — %s", ErrWhisperAPIError, resp.StatusCode, string(respBody))
	}

	c.logger.Info("Whisper transcription complete", "file", filepath.Base(audioPath), "srt_bytes", len(respBody))

	return string(respBody), nil
}

// NeedsChunking returns true if the audio file exceeds the Whisper API size limit.
func NeedsChunking(audioPath string) (bool, error) {
	info, err := os.Stat(audioPath)
	if err != nil {
		return false, fmt.Errorf("stat audio file: %w", err)
	}
	return info.Size() > WhisperMaxFileSize, nil
}

// SplitAudioChunks splits a WAV file into chunks of WhisperChunkDuration seconds.
// Returns paths to the chunk files. Caller is responsible for cleanup.
func SplitAudioChunks(ctx context.Context, audioPath string) ([]string, error) {
	// Get duration from WAV header (16kHz, mono, 16-bit PCM)
	duration, err := getWAVDuration(audioPath)
	if err != nil {
		return nil, fmt.Errorf("get audio duration: %w", err)
	}

	if duration <= WhisperChunkDuration {
		return []string{audioPath}, nil
	}

	var chunks []string
	for start := 0; start < int(duration); start += WhisperChunkDuration {
		chunkFile, err := os.CreateTemp("", fmt.Sprintf("vido-chunk-%d-*.wav", start))
		if err != nil {
			// Cleanup already created chunks
			for _, c := range chunks {
				os.Remove(c)
			}
			return nil, fmt.Errorf("create chunk temp file: %w", err)
		}
		chunkPath := chunkFile.Name()
		chunkFile.Close()

		//nolint:gosec // audioPath comes from our own temp extraction
		cmd := execCommandContext(ctx, "ffmpeg",
			"-i", audioPath,
			"-ss", fmt.Sprintf("%d", start),
			"-t", fmt.Sprintf("%d", WhisperChunkDuration),
			"-acodec", "pcm_s16le",
			"-ar", "16000",
			"-ac", "1",
			"-y",
			chunkPath,
		)

		if output, err := cmd.CombinedOutput(); err != nil {
			for _, c := range chunks {
				os.Remove(c)
			}
			os.Remove(chunkPath)
			return nil, fmt.Errorf("ffmpeg chunk split at %ds: %w — %s", start, err, string(output))
		}

		chunks = append(chunks, chunkPath)
	}

	return chunks, nil
}

// execCommandContext wraps exec.CommandContext to allow testing
var execCommandContext = execCommandContextReal

func execCommandContextReal(ctx context.Context, name string, args ...string) command {
	return execCmd{exec.CommandContext(ctx, name, args...)}
}

// command interface for testing
type command interface {
	CombinedOutput() ([]byte, error)
}

type execCmd struct {
	*exec.Cmd
}

// MergeSRTChunks merges multiple SRT strings from chunked transcription, adjusting timestamps.
func MergeSRTChunks(chunks []string, chunkDuration int) string {
	if len(chunks) == 0 {
		return ""
	}
	if len(chunks) == 1 {
		return chunks[0]
	}

	var merged bytes.Buffer
	seqNum := 1

	for i, chunk := range chunks {
		offsetSeconds := i * chunkDuration
		adjusted := adjustSRTTimestamps(chunk, offsetSeconds, &seqNum)
		merged.WriteString(adjusted)
	}

	return merged.String()
}

// getWAVDuration calculates duration from WAV file header (PCM 16kHz mono 16-bit).
func getWAVDuration(path string) (float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// Read RIFF header
	header := make([]byte, 44)
	if _, err := io.ReadFull(f, header); err != nil {
		return 0, fmt.Errorf("read WAV header: %w", err)
	}

	// Validate RIFF/WAVE
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return 0, fmt.Errorf("not a WAV file")
	}

	// Get data chunk size (bytes 40-43) and byte rate (bytes 28-31)
	byteRate := binary.LittleEndian.Uint32(header[28:32])
	dataSize := binary.LittleEndian.Uint32(header[40:44])

	if byteRate == 0 {
		return 0, fmt.Errorf("invalid WAV byte rate")
	}

	return float64(dataSize) / float64(byteRate), nil
}

// adjustSRTTimestamps adjusts SRT timestamp lines by an offset and renumbers sequences.
func adjustSRTTimestamps(srt string, offsetSeconds int, seqNum *int) string {
	if offsetSeconds == 0 && *seqNum == 1 {
		// First chunk, no adjustment needed; just count sequences
		result := &bytes.Buffer{}
		lines := splitLines(srt)
		for i := 0; i < len(lines); i++ {
			line := lines[i]
			// Check if this is a sequence number line (digits only, followed by timestamp line)
			if isSequenceNumber(line) && i+1 < len(lines) && isTimestampLine(lines[i+1]) {
				fmt.Fprintf(result, "%d\n", *seqNum)
				*seqNum++
				continue
			}
			result.WriteString(line)
			result.WriteByte('\n')
		}
		return result.String()
	}

	result := &bytes.Buffer{}
	lines := splitLines(srt)
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if isSequenceNumber(line) && i+1 < len(lines) && isTimestampLine(lines[i+1]) {
			fmt.Fprintf(result, "%d\n", *seqNum)
			*seqNum++
			continue
		}
		if isTimestampLine(line) {
			adjusted := offsetTimestampLine(line, offsetSeconds)
			result.WriteString(adjusted)
			result.WriteByte('\n')
			continue
		}
		result.WriteString(line)
		result.WriteByte('\n')
	}
	return result.String()
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func isSequenceNumber(line string) bool {
	if len(line) == 0 {
		return false
	}
	for _, c := range line {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isTimestampLine(line string) bool {
	// SRT timestamp format: 00:00:00,000 --> 00:00:00,000
	return len(line) >= 29 && line[2] == ':' && line[5] == ':' && line[8] == ','
}

func offsetTimestampLine(line string, offsetSeconds int) string {
	// Parse: 00:00:00,000 --> 00:00:00,000
	if len(line) < 29 {
		return line
	}

	start := parseSRTTimestamp(line[0:12])
	end := parseSRTTimestamp(line[17:29])

	start += offsetSeconds * 1000
	end += offsetSeconds * 1000

	return fmt.Sprintf("%s --> %s", formatSRTTimestamp(start), formatSRTTimestamp(end))
}

// parseSRTTimestamp parses "HH:MM:SS,mmm" to milliseconds.
func parseSRTTimestamp(ts string) int {
	if len(ts) < 12 {
		return 0
	}
	h := int(ts[0]-'0')*10 + int(ts[1]-'0')
	m := int(ts[3]-'0')*10 + int(ts[4]-'0')
	s := int(ts[6]-'0')*10 + int(ts[7]-'0')
	ms := int(ts[9]-'0')*100 + int(ts[10]-'0')*10 + int(ts[11]-'0')
	return h*3600000 + m*60000 + s*1000 + ms
}

// formatSRTTimestamp formats milliseconds to "HH:MM:SS,mmm".
func formatSRTTimestamp(ms int) string {
	h := ms / 3600000
	ms %= 3600000
	m := ms / 60000
	ms %= 60000
	s := ms / 1000
	ms %= 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}
