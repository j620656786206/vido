package ai

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhisperClient_Transcribe_Success(t *testing.T) {
	expectedSRT := "1\n00:00:01,000 --> 00:00:03,000\nHello world\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer test-key")
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

		err := r.ParseMultipartForm(32 << 20)
		require.NoError(t, err)
		assert.Equal(t, "whisper-1", r.FormValue("model"))
		assert.Equal(t, "srt", r.FormValue("response_format"))

		_, header, err := r.FormFile("file")
		require.NoError(t, err)
		assert.NotEmpty(t, header.Filename)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedSRT))
	}))
	defer server.Close()

	client := NewWhisperClient("test-key",
		WithWhisperBaseURL(server.URL),
		WithWhisperTimeout(10*time.Second),
	)

	// Create a temp audio file
	tmpFile, err := os.CreateTemp("", "whisper-test-*.wav")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("fake audio data"))
	tmpFile.Close()

	srt, err := client.Transcribe(context.Background(), tmpFile.Name())
	require.NoError(t, err)
	assert.Equal(t, expectedSRT, srt)
}

func TestWhisperClient_Transcribe_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"message": "invalid audio"}}`))
	}))
	defer server.Close()

	client := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))

	tmpFile, err := os.CreateTemp("", "whisper-test-*.wav")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("fake"))
	tmpFile.Close()

	_, err = client.Transcribe(context.Background(), tmpFile.Name())
	assert.ErrorIs(t, err, ErrWhisperAPIError)
}

func TestWhisperClient_Transcribe_NoAPIKey(t *testing.T) {
	client := NewWhisperClient("")
	_, err := client.Transcribe(context.Background(), "/tmp/test.wav")
	assert.ErrorIs(t, err, ErrWhisperNotConfigured)
}

func TestWhisperClient_Transcribe_FileNotFound(t *testing.T) {
	client := NewWhisperClient("test-key")
	_, err := client.Transcribe(context.Background(), "/nonexistent/file.wav")
	assert.Error(t, err)
}

func TestNeedsChunking(t *testing.T) {
	// Small file — no chunking
	small, err := os.CreateTemp("", "small-*.wav")
	require.NoError(t, err)
	defer os.Remove(small.Name())
	small.Write(make([]byte, 1024))
	small.Close()

	needs, err := NeedsChunking(small.Name())
	require.NoError(t, err)
	assert.False(t, needs)

	// File not found
	_, err = NeedsChunking("/nonexistent.wav")
	assert.Error(t, err)
}

func TestParseSRTTimestamp(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"00:00:00,000", 0},
		{"00:00:01,000", 1000},
		{"00:01:00,000", 60000},
		{"01:00:00,000", 3600000},
		{"01:23:45,678", 5025678},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseSRTTimestamp(tt.input))
		})
	}
}

func TestFormatSRTTimestamp(t *testing.T) {
	tests := []struct {
		ms       int
		expected string
	}{
		{0, "00:00:00,000"},
		{1000, "00:00:01,000"},
		{60000, "00:01:00,000"},
		{3600000, "01:00:00,000"},
		{5025678, "01:23:45,678"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatSRTTimestamp(tt.ms))
		})
	}
}

func TestMergeSRTChunks_SingleChunk(t *testing.T) {
	srt := "1\n00:00:01,000 --> 00:00:03,000\nHello\n\n"
	result := MergeSRTChunks([]string{srt}, WhisperChunkDuration)
	assert.Equal(t, srt, result)
}

func TestMergeSRTChunks_Empty(t *testing.T) {
	result := MergeSRTChunks(nil, WhisperChunkDuration)
	assert.Equal(t, "", result)
}

func TestMergeSRTChunks_TwoChunks(t *testing.T) {
	chunk1 := "1\n00:00:01,000 --> 00:00:03,000\nHello\n\n2\n00:00:05,000 --> 00:00:08,000\nWorld\n\n"
	chunk2 := "1\n00:00:01,000 --> 00:00:04,000\nFoo\n\n"

	result := MergeSRTChunks([]string{chunk1, chunk2}, 600)

	// chunk1: seqs 1,2 with original timestamps
	assert.Contains(t, result, "1\n00:00:01,000 --> 00:00:03,000\nHello\n")
	assert.Contains(t, result, "2\n00:00:05,000 --> 00:00:08,000\nWorld\n")
	// chunk2: seq 3, timestamps offset by 600s (10 min)
	assert.Contains(t, result, "3\n00:10:01,000 --> 00:10:04,000\nFoo\n")
}

func TestIsTimestampLine(t *testing.T) {
	assert.True(t, isTimestampLine("00:00:01,000 --> 00:00:03,000"))
	assert.True(t, isTimestampLine("01:23:45,678 --> 02:34:56,789"))
	assert.False(t, isTimestampLine("1"))
	assert.False(t, isTimestampLine("Hello world"))
	assert.False(t, isTimestampLine(""))
}

func TestIsSequenceNumber(t *testing.T) {
	assert.True(t, isSequenceNumber("1"))
	assert.True(t, isSequenceNumber("123"))
	assert.False(t, isSequenceNumber(""))
	assert.False(t, isSequenceNumber("abc"))
	assert.False(t, isSequenceNumber("1a"))
}

func TestOffsetTimestampLine(t *testing.T) {
	line := "00:00:01,000 --> 00:00:03,000"
	result := offsetTimestampLine(line, 600)
	assert.Equal(t, "00:10:01,000 --> 00:10:03,000", result)
}

func TestGetWAVDuration(t *testing.T) {
	// Create a minimal valid WAV file (16kHz, mono, 16-bit PCM)
	tmpFile, err := os.CreateTemp("", "wav-test-*.wav")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write WAV header for 1 second of audio (16kHz * 2 bytes * 1 channel = 32000 bytes data)
	sampleRate := uint32(16000)
	bitsPerSample := uint16(16)
	channels := uint16(1)
	byteRate := sampleRate * uint32(channels) * uint32(bitsPerSample/8)
	blockAlign := channels * (bitsPerSample / 8)
	dataSize := uint32(32000) // 1 second

	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], 36+dataSize)
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16) // fmt chunk size
	binary.LittleEndian.PutUint16(header[20:22], 1)  // PCM
	binary.LittleEndian.PutUint16(header[22:24], channels)
	binary.LittleEndian.PutUint32(header[24:28], sampleRate)
	binary.LittleEndian.PutUint32(header[28:32], byteRate)
	binary.LittleEndian.PutUint16(header[32:34], blockAlign)
	binary.LittleEndian.PutUint16(header[34:36], bitsPerSample)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], dataSize)

	tmpFile.Write(header)
	tmpFile.Write(make([]byte, dataSize))
	tmpFile.Close()

	duration, err := getWAVDuration(tmpFile.Name())
	require.NoError(t, err)
	assert.InDelta(t, 1.0, duration, 0.01, "expected ~1 second duration")
}

func TestGetWAVDuration_InvalidFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "invalid-*.wav")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("not a wav file at all"))
	tmpFile.Close()

	_, err = getWAVDuration(tmpFile.Name())
	assert.Error(t, err)
}

func TestGetWAVDuration_FileNotFound(t *testing.T) {
	_, err := getWAVDuration("/nonexistent.wav")
	assert.Error(t, err)
}

func TestSplitAudioChunks_SmallFile(t *testing.T) {
	// Create a WAV file shorter than chunk duration — should return original path
	tmpFile, err := os.CreateTemp("", "short-*.wav")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write WAV header for 10 seconds (well under chunk duration)
	dataSize := uint32(320000) // 10 seconds at 16kHz mono 16-bit
	writeTestWAV(t, tmpFile, dataSize)

	chunks, err := SplitAudioChunks(context.Background(), tmpFile.Name())
	require.NoError(t, err)
	require.Len(t, chunks, 1)
	assert.Equal(t, tmpFile.Name(), chunks[0])
}

func writeTestWAV(t *testing.T, f *os.File, dataSize uint32) {
	t.Helper()
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], 36+dataSize)
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1)     // PCM
	binary.LittleEndian.PutUint16(header[22:24], 1)     // mono
	binary.LittleEndian.PutUint32(header[24:28], 16000) // sample rate
	binary.LittleEndian.PutUint32(header[28:32], 32000) // byte rate
	binary.LittleEndian.PutUint16(header[32:34], 2)     // block align
	binary.LittleEndian.PutUint16(header[34:36], 16)    // bits per sample
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], dataSize)
	f.Write(header)
	// Write minimal data (not full dataSize to save time; header is what matters for duration calc)
	f.Write(make([]byte, min(dataSize, 1024)))
	f.Close()
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func TestWhisperClient_Transcribe_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": {"message": "rate limit exceeded"}}`))
	}))
	defer server.Close()

	client := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))

	tmpFile, err := os.CreateTemp("", "whisper-rate-*.wav")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("fake"))
	tmpFile.Close()

	_, err = client.Transcribe(context.Background(), tmpFile.Name())
	assert.ErrorIs(t, err, ErrWhisperAPIError)
	assert.Contains(t, err.Error(), "429")
}

func TestWhisperClient_Transcribe_AlreadyCancelledContext(t *testing.T) {
	// Pre-cancelled context should fail immediately without hitting server
	client := NewWhisperClient("test-key",
		WithWhisperBaseURL("http://127.0.0.1:1"), // unreachable, but context is already done
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	tmpFile, err := os.CreateTemp("", "whisper-cancel-*.wav")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("fake"))
	tmpFile.Close()

	_, err = client.Transcribe(ctx, tmpFile.Name())
	assert.Error(t, err)
}

func TestMergeSRTChunks_ThreeChunks(t *testing.T) {
	chunk1 := "1\n00:00:01,000 --> 00:00:03,000\nA\n\n"
	chunk2 := "1\n00:00:02,000 --> 00:00:05,000\nB\n\n"
	chunk3 := "1\n00:00:01,500 --> 00:00:04,000\nC\n\n"

	result := MergeSRTChunks([]string{chunk1, chunk2, chunk3}, 600)

	// chunk1: seq 1, no offset
	assert.Contains(t, result, "1\n00:00:01,000 --> 00:00:03,000\nA\n")
	// chunk2: seq 2, +600s offset
	assert.Contains(t, result, "2\n00:10:02,000 --> 00:10:05,000\nB\n")
	// chunk3: seq 3, +1200s offset
	assert.Contains(t, result, "3\n00:20:01,500 --> 00:20:04,000\nC\n")
}

func TestSplitAudioChunks_InvalidWAV(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "bad-*.wav")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	fmt.Fprint(tmpFile, "not a wav")
	tmpFile.Close()

	_, err = SplitAudioChunks(context.Background(), tmpFile.Name())
	assert.Error(t, err)
}
