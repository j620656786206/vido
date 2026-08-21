package ai

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
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
		// 9R-5 DELIBERATE CONTRACT CHANGE (was "srt", story 9-2a task 2.3):
		// only verbose_json carries the per-segment confidence the
		// hallucination filter judges on. The SRT the caller receives is now
		// rendered here from those segments — see segmentsToSRT.
		assert.Equal(t, "verbose_json", r.FormValue("response_format"))

		_, header, err := r.FormFile("file")
		require.NoError(t, err)
		assert.NotEmpty(t, header.Filename)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"language":"english","duration":3.0,"segments":[` +
			`{"id":0,"start":1.0,"end":3.0,"text":"Hello world",` +
			`"avg_logprob":-0.2,"compression_ratio":1.2,"no_speech_prob":0.01}]}`))
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

	chunks, chunkSeconds, err := SplitAudioChunks(context.Background(), tmpFile.Name())
	require.NoError(t, err)
	require.Len(t, chunks, 1)
	assert.Equal(t, tmpFile.Name(), chunks[0])
	assert.Equal(t, WhisperChunkDuration, chunkSeconds)
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

func TestParseSRTTimestamp_MalformedInput(t *testing.T) {
	// Non-digit characters should return 0 instead of garbage
	assert.Equal(t, 0, parseSRTTimestamp("XX:XX:XX,XXX"))
	assert.Equal(t, 0, parseSRTTimestamp("ab:cd:ef,ghi"))
	// Short input
	assert.Equal(t, 0, parseSRTTimestamp("00:00"))
	// Empty
	assert.Equal(t, 0, parseSRTTimestamp(""))
}

func TestSplitAudioChunks_InvalidWAV(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "bad-*.wav")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	fmt.Fprint(tmpFile, "not a wav")
	tmpFile.Close()

	_, _, err = SplitAudioChunks(context.Background(), tmpFile.Name())
	assert.Error(t, err)
}

// --- 9R-2: language pinned from the audio track ---

func TestWhisperClient_TranscribeWithLanguage_MultipartCarriesLanguage(t *testing.T) {
	var gotLang string
	var hadLangField bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		vals := r.MultipartForm.Value["language"]
		hadLangField = len(vals) > 0
		if hadLangField {
			gotLang = vals[0]
		}
		w.Write([]byte("1\n00:00:00,000 --> 00:00:01,000\nhello\n"))
	}))
	defer server.Close()

	audio := writeTempAudio(t)

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))

	// AC #3: given a track lang, the multipart body includes language=<iso2>.
	_, err := c.TranscribeWithLanguage(context.Background(), audio, "en")
	if err != nil {
		t.Fatalf("transcribe: %v", err)
	}
	if !hadLangField || gotLang != "en" {
		t.Fatalf("expected language=en in multipart body, got present=%v value=%q", hadLangField, gotLang)
	}

	// AC #2: empty lang (und track) => auto-detect, NO language field.
	hadLangField = false
	gotLang = ""
	_, err = c.TranscribeWithLanguage(context.Background(), audio, "")
	if err != nil {
		t.Fatalf("transcribe (auto): %v", err)
	}
	if hadLangField {
		t.Fatalf("expected NO language field for auto-detect, got %q", gotLang)
	}
}

func writeTempAudio(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "audio-*.wav")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("RIFFfakewavdata"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

// --- 9R-3: chunking 413 fix ---

// writeWAVWithChunks writes a WAV whose header contains extra RIFF chunks
// (as ffmpeg emits) between fmt and data — the shape the old fixed-offset
// parser misread (the 413 root cause).
func writeWAVWithChunks(t *testing.T, f *os.File, byteRate uint32, dataSize uint32, extraChunk bool) {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // riff size (unused by parser)
	buf.WriteString("WAVE")
	// fmt chunk (16 bytes, PCM)
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // mono
	binary.Write(&buf, binary.LittleEndian, uint32(16000))
	binary.Write(&buf, binary.LittleEndian, byteRate)
	binary.Write(&buf, binary.LittleEndian, uint16(2))
	binary.Write(&buf, binary.LittleEndian, uint16(16))
	if extraChunk {
		// LIST/INFO chunk between fmt and data — fixed-offset readers now see
		// garbage at bytes 40:44.
		info := []byte("INFOISFT\x0e\x00\x00\x00Lavf61.1.100\x00\x00")
		buf.WriteString("LIST")
		binary.Write(&buf, binary.LittleEndian, uint32(len(info)))
		buf.Write(info)
	}
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataSize)
	if _, err := f.Write(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	// Write (sparse-ish) data payload.
	if _, err := f.Write(make([]byte, dataSize)); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestParseWAVInfo_HeaderRobust_ExtraChunks(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "list-*.wav")
	require.NoError(t, err)
	byteRate := uint32(32000)      // 16kHz mono 16-bit
	dataSize := uint32(32000 * 30) // 30 seconds
	writeWAVWithChunks(t, f, byteRate, dataSize, true)

	duration, br, err := parseWAVInfo(f.Name())
	require.NoError(t, err)
	assert.Equal(t, byteRate, br)
	assert.InDelta(t, 30.0, duration, 0.01,
		"duration must be parsed correctly even with a LIST chunk before data (old fixed-offset read returned garbage)")
}

func TestNeedsChunking_UsesHeadroomBudget(t *testing.T) {
	// A file between the 24MiB budget and the 25MiB API limit MUST chunk —
	// file bytes + multipart overhead would otherwise 413.
	f, err := os.CreateTemp(t.TempDir(), "mid-*.wav")
	require.NoError(t, err)
	require.NoError(t, f.Truncate(WhisperChunkTargetBytes+1024))
	f.Close()

	needs, err := NeedsChunking(f.Name())
	require.NoError(t, err)
	assert.True(t, needs)
}

func TestSplitAudioChunks_LargeFile_ChunksUnderLimit(t *testing.T) {
	// >25MiB WAV (AC #3): expect N chunks, each under the limit, and the
	// returned chunkSeconds driving contiguous merged timestamps.
	f, err := os.CreateTemp(t.TempDir(), "big-*.wav")
	require.NoError(t, err)
	byteRate := uint32(32000)
	dataSize := uint32(26 * 1024 * 1024) // ~852s at 32KB/s → 2 chunks of 600s
	writeWAVWithChunks(t, f, byteRate, dataSize, true)

	// Stub ffmpeg: write a small fake chunk file per call.
	origExec := execCommandContext
	var ffmpegCalls [][]string
	execCommandContext = func(ctx context.Context, name string, args ...string) command {
		ffmpegCalls = append(ffmpegCalls, append([]string{name}, args...))
		return fakeChunkCmd{args: args}
	}
	defer func() { execCommandContext = origExec }()

	chunks, chunkSeconds, err := SplitAudioChunks(context.Background(), f.Name())
	require.NoError(t, err)
	defer func() {
		for _, c := range chunks {
			os.Remove(c)
		}
	}()

	assert.Equal(t, WhisperChunkDuration, chunkSeconds)
	require.Len(t, chunks, 2, "852s at 600s chunks → 2 chunks")
	for _, c := range chunks {
		info, err := os.Stat(c)
		require.NoError(t, err)
		assert.Less(t, info.Size(), int64(WhisperMaxFileSize))
	}
	// ffmpeg must be invoked with -t <chunkSeconds> and contiguous -ss offsets.
	require.Len(t, ffmpegCalls, 2)
	assert.Contains(t, ffmpegCalls[0], "-ss")
	assert.Contains(t, ffmpegCalls[0], "0")
	assert.Contains(t, ffmpegCalls[1], "600")

	// Contiguous timestamps on merge, driven by the RETURNED chunkSeconds.
	merged := MergeSRTChunks([]string{
		"1\n00:00:00,000 --> 00:00:01,000\nfirst\n\n",
		"1\n00:00:00,000 --> 00:00:01,000\nsecond\n\n",
	}, chunkSeconds)
	assert.Contains(t, merged, "00:00:00,000 --> 00:00:01,000")
	assert.Contains(t, merged, "00:10:00,000 --> 00:10:01,000",
		"second chunk must be offset by exactly chunkSeconds")
}

func TestSplitAudioChunks_HighByteRate_ShrinksChunkSeconds(t *testing.T) {
	// 48kHz stereo (192KB/s): 600s would be ~115MiB per chunk → chunkSeconds
	// must shrink so duration*byteRate stays under the 24MiB budget.
	f, err := os.CreateTemp(t.TempDir(), "hbr-*.wav")
	require.NoError(t, err)
	byteRate := uint32(192000)
	dataSize := uint32(192000 * 10) // 10s — small file, no split needed
	writeWAVWithChunks(t, f, byteRate, dataSize, false)

	chunks, chunkSeconds, err := SplitAudioChunks(context.Background(), f.Name())
	require.NoError(t, err)
	require.Len(t, chunks, 1)
	assert.LessOrEqual(t, chunkSeconds, int(uint32(WhisperChunkTargetBytes)/byteRate))
	assert.Less(t, chunkSeconds, WhisperChunkDuration)
}

type fakeChunkCmd struct {
	args []string
}

func (c fakeChunkCmd) CombinedOutput() ([]byte, error) {
	// Last arg is the chunk output path — write a small fake WAV there.
	out := c.args[len(c.args)-1]
	return nil, os.WriteFile(out, []byte("RIFFfake"), 0o644)
}

// --- 9R-11: Whisper honors governor + budget ---

func TestWhisperClient_BudgetCutoffStopsTranscribe(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte("1\n00:00:00,000 --> 00:00:01,000\nx\n"))
	}))
	defer server.Close()

	audio := writeTempAudio(t)
	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	b := NewBudget(0.001)
	b.RecordASR(600) // 10 min * $0.006 = $0.06 → over ceiling
	ctx := WithBudget(context.Background(), b)

	_, err := c.Transcribe(ctx, audio)
	require.ErrorIs(t, err, ErrBudgetExceeded)
	assert.Equal(t, int32(0), hits.Load())
}

// --- 9R-9: ASRProvider + configurable engine/base-URL ---

func TestWhisperClient_SatisfiesASRProvider(t *testing.T) {
	var _ ASRProvider = NewWhisperClient("k")
}

func TestWhisperClient_ConfigurableBaseURLAndModel(t *testing.T) {
	// Smoke test against a mock OpenAI-compatible /v1/audio/transcriptions
	// server at a custom base URL with a self-hosted engine model id.
	var gotModel string
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = r.ParseMultipartForm(32 << 20)
		if v := r.MultipartForm.Value["model"]; len(v) > 0 {
			gotModel = v[0]
		}
		w.Write([]byte("1\n00:00:00,000 --> 00:00:01,000\nhello\n"))
	}))
	defer server.Close()

	audio := writeTempAudio(t)
	c := NewWhisperClient("test-key",
		WithWhisperBaseURL(server.URL+"/v1/audio/transcriptions"),
		WithWhisperModel("Systran/faster-whisper-small"),
	)

	// Use the interface, not the concrete type — proves decoupling.
	var provider ASRProvider = c
	srt, err := provider.Transcribe(context.Background(), audio)
	require.NoError(t, err)
	assert.Contains(t, srt, "hello")
	assert.Equal(t, "Systran/faster-whisper-small", gotModel, "self-hosted engine model id must be sent")
	assert.Equal(t, "/v1/audio/transcriptions", gotPath)
}

func TestWhisperClient_DefaultModelIsWhisper1(t *testing.T) {
	var gotModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(32 << 20)
		if v := r.MultipartForm.Value["model"]; len(v) > 0 {
			gotModel = v[0]
		}
		w.Write([]byte("1\n00:00:00,000 --> 00:00:01,000\nx\n"))
	}))
	defer server.Close()

	c := NewWhisperClient("k", WithWhisperBaseURL(server.URL))
	_, err := c.Transcribe(context.Background(), writeTempAudio(t))
	require.NoError(t, err)
	assert.Equal(t, WhisperModel, gotModel, "default model stays whisper-1")
}

// --- sub-5-1 AC #1: metering follows the endpoint the client actually calls ---

// transcribeWithParseableWAV runs one transcription of a REAL parseable WAV
// (parseWAVInfo must succeed for RecordASR* to fire) against a mock server.
func transcribeWithParseableWAV(t *testing.T, b *Budget, opts ...WhisperOption) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("1\n00:00:00,000 --> 00:00:01,000\nhello\n"))
	}))
	defer server.Close()

	f, err := os.CreateTemp(t.TempDir(), "metered-*.wav")
	require.NoError(t, err)
	byteRate := uint32(32000)
	writeWAVWithChunks(t, f, byteRate, byteRate*60, false) // exactly 60s of audio

	c := NewWhisperClient("test-key", append([]WhisperOption{WithWhisperBaseURL(server.URL)}, opts...)...)
	_, err = c.Transcribe(WithBudget(context.Background(), b), f.Name())
	require.NoError(t, err)
}

func TestWhisperClient_SelfHostedEndpointRecordsZeroSpend(t *testing.T) {
	// The base URL override IS the self-hosted signal (c.baseURL != WhisperAPIURL)
	// — the httptest server plays the role of a local faster-whisper/Speaches.
	b := NewBudget(0)
	transcribeWithParseableWAV(t, b)
	snap := b.Snapshot()
	assert.Zero(t, snap.SpentUSD, "self-hosted ASR must not bill the hosted per-minute rate")
	assert.InDelta(t, 60, snap.ASRSeconds, 1.0, "the audio minutes are still metered")
	assert.Equal(t, 1, snap.ASRCalls)
}

func TestWhisperClient_IsSelfHosted_ReadsTheActualEndpoint(t *testing.T) {
	assert.False(t, NewWhisperClient("k").isSelfHosted(),
		"default base URL = the paid hosted API")
	assert.True(t, NewWhisperClient("k", WithWhisperBaseURL("http://nas.local:8000/v1/audio/transcriptions")).isSelfHosted())
}

func TestWhisperClient_MeterRateIsSameSourceAsEstimator(t *testing.T) {
	// 同源斷言 (hosted side): a client at the DEFAULT endpoint meters exactly
	// EstimatedASRPerMinuteUSD(false) per minute. The call site passes
	// EstimatedASRPerMinuteUSD(c.isSelfHosted()) — this pins the hosted branch
	// without hitting the real API by computing what one minute would record.
	b := NewBudget(0)
	b.RecordASRWithRate(60, EstimatedASRPerMinuteUSD(NewWhisperClient("k").isSelfHosted()))
	assert.InDelta(t, HostedASRPerMinuteUSD(), b.SpentUSD(), 1e-9)
}

// CR M1 (sub-5-1): quote and invoice derive their self-hosted answer from ONE
// predicate. For every configured ASR_BASE_URL value, the estimate side's
// IsSelfHostedASRBaseURL must agree with the metering side's client-derived
// judgment — including the trap value: the official endpoint set explicitly.
func TestSelfHostedJudgment_SingleSource(t *testing.T) {
	for _, tc := range []struct {
		name       string
		configured string // ASR_BASE_URL as the operator set it ("" = unset)
		selfHosted bool
	}{
		{"unset → hosted", "", false},
		{"official endpoint set explicitly → still hosted", WhisperAPIURL, false},
		{"custom endpoint → self-hosted", "http://nas.local:8000/v1/audio/transcriptions", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The estimate side (main.go → NewGenerationCandidateService).
			assert.Equal(t, tc.selfHosted, IsSelfHostedASRBaseURL(tc.configured))

			// The metering side: build the client exactly as main.go does —
			// override only when the config value is non-empty.
			opts := []WhisperOption{}
			if tc.configured != "" {
				opts = append(opts, WithWhisperBaseURL(tc.configured))
			}
			assert.Equal(t, tc.selfHosted, NewWhisperClient("k", opts...).isSelfHosted(),
				"estimate and metering disagreeing is a quote the invoice contradicts")
		})
	}
}

// --- sub-5-2 AC #3: a self-hosted engine authenticates nothing --------------

// authProbeServer records whether an Authorization header reached the engine.
func authProbeServer(t *testing.T, seen *string, present *bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = r.Header.Get("Authorization")
		_, *present = r.Header["Authorization"]
		w.Write([]byte("1\n00:00:00,000 --> 00:00:01,000\nhello\n"))
	}))
}

func wavFixture(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "asr-*.wav")
	require.NoError(t, err)
	byteRate := uint32(32000)
	writeWAVWithChunks(t, f, byteRate, byteRate*2, false)
	return f.Name()
}

// Before sub-5-2 a keyless client short-circuited on ErrWhisperNotConfigured, so
// a self-hosted deployment (which needs no key) had to invent a dummy
// OPENAI_API_KEY to get any ASR at all — while sub-5-1 already metered it at $0
// and the docs advertised it as supported.
func TestWhisperClient_SelfHostedTranscribesWithoutAKey(t *testing.T) {
	var seen string
	var present bool
	server := authProbeServer(t, &seen, &present)
	defer server.Close()

	c := NewWhisperClient("", WithWhisperBaseURL(server.URL))
	srt, err := c.Transcribe(context.Background(), wavFixture(t))

	require.NoError(t, err)
	assert.Contains(t, srt, "hello")
	assert.False(t, present,
		"an empty key must omit the header entirely — `Bearer ` with no value is a malformed credential some engines reject")
}

// The paid hosted API still costs money and still needs a key: the AC #3
// relaxation must not leak into that path.
func TestWhisperClient_HostedStillRequiresAKey(t *testing.T) {
	c := NewWhisperClient("")

	_, err := c.Transcribe(context.Background(), wavFixture(t))

	require.ErrorIs(t, err, ErrWhisperNotConfigured)
}

// A self-hosted engine MAY still be key-protected (a reverse proxy, a shared
// LAN). When a key resolves, it is sent exactly as before.
func TestWhisperClient_SelfHostedWithAKeyStillSendsIt(t *testing.T) {
	var seen string
	var present bool
	server := authProbeServer(t, &seen, &present)
	defer server.Close()

	c := NewWhisperClient("sk-lan", WithWhisperBaseURL(server.URL))
	_, err := c.Transcribe(context.Background(), wavFixture(t))

	require.NoError(t, err)
	require.True(t, present)
	assert.Equal(t, "Bearer sk-lan", seen)
}

// --- 9R-5: verbose_json request, hallucination filter, sticky SRT fallback ---

const verboseJSONOneCue = `{"language":"english","duration":3.0,"segments":[` +
	`{"id":0,"start":1.0,"end":3.0,"text":"Hello world",` +
	`"avg_logprob":-0.2,"compression_ratio":1.2,"no_speech_prob":0.01}]}`

// formatRecorder captures the response_format of every request the client makes,
// so a test can assert the exact request SEQUENCE rather than just the outcome.
type formatRecorder struct {
	mu      sync.Mutex
	formats []string
}

func (fr *formatRecorder) record(t *testing.T, r *http.Request) string {
	t.Helper()
	require.NoError(t, r.ParseMultipartForm(32<<20))
	f := r.FormValue("response_format")
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.formats = append(fr.formats, f)
	return f
}

func (fr *formatRecorder) seen() []string {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	return append([]string(nil), fr.formats...)
}

// An engine that REJECTS verbose_json must still produce subtitles: the
// verbose_json `text` field carries no timestamps, so "no segments" can never
// be allowed to mean "no subtitle".
func TestWhisperClient_RejectedVerboseFallsBackToSRTAndLatches(t *testing.T) {
	fallbackSRT := "1\n00:00:01,000 --> 00:00:03,000\nHello world\n\n"
	rec := &formatRecorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rec.record(t, r) == "verbose_json" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"message":"unsupported response_format"}}`))
			return
		}
		w.Write([]byte(fallbackSRT))
	}))
	defer server.Close()

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	audio := writeTempAudio(t)

	srt, err := c.Transcribe(context.Background(), audio)
	require.NoError(t, err)
	assert.Equal(t, fallbackSRT, srt)

	// A second call must NOT probe again — a ten-chunk file would otherwise
	// pay ten pointless round trips.
	srt2, err := c.Transcribe(context.Background(), audio)
	require.NoError(t, err)
	assert.Equal(t, fallbackSRT, srt2)

	assert.Equal(t, []string{"verbose_json", "srt", "srt"}, rec.seen(),
		"probe once, then stay on srt for the life of the client")
}

// A 200 that is not verbose_json (an engine echoing SRT back regardless of the
// requested format) is the same class of failure as an outright rejection.
func TestWhisperClient_UnparseableVerboseBodyFallsBack(t *testing.T) {
	srtBody := "1\n00:00:01,000 --> 00:00:02,000\nEchoed\n\n"
	rec := &formatRecorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(t, r)
		w.Write([]byte(srtBody))
	}))
	defer server.Close()

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	srt, err := c.Transcribe(context.Background(), writeTempAudio(t))
	require.NoError(t, err)
	assert.Equal(t, srtBody, srt)
	assert.Equal(t, []string{"verbose_json", "srt"}, rec.seen())
}

// A transcript with no segment array = the engine served the wrong JSON shape.
// The words exist but nothing says WHEN, so recover a cueable transcript.
func TestWhisperClient_TextWithoutSegmentsFallsBack(t *testing.T) {
	rec := &formatRecorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rec.record(t, r) == "verbose_json" {
			w.Write([]byte(`{"text":"Hello there, this is real dialogue","segments":[]}`))
			return
		}
		w.Write([]byte("1\n00:00:01,000 --> 00:00:02,000\nFrom srt\n\n"))
	}))
	defer server.Close()

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	srt, err := c.Transcribe(context.Background(), writeTempAudio(t))
	require.NoError(t, err)
	assert.Contains(t, srt, "From srt")
	assert.Equal(t, []string{"verbose_json", "srt"}, rec.seen())
}

// 🔴 CR H1 REGRESSION GUARD. A genuinely SILENT chunk answers with an empty
// segment array and an empty transcript — ten minutes of credits, the single
// most on-topic input this story has. Treating that as a broken engine latched
// the fallback and killed hallucination filtering for every later chunk AND
// every later media item, while paying a wasted srt request each time.
func TestWhisperClient_SilentChunkDoesNotLatchTheFallback(t *testing.T) {
	rec := &formatRecorder{}
	silent := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rec.record(t, r) == "verbose_json" && silent {
			w.Write([]byte(`{"task":"transcribe","language":"english","duration":600.0,"text":"","segments":[]}`))
			return
		}
		w.Write([]byte(verboseJSONOneCue))
	}))
	defer server.Close()

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	audio := writeTempAudio(t)

	detail, err := c.TranscribeDetailed(context.Background(), audio, "en")
	require.NoError(t, err)
	assert.Equal(t, "", detail.SRT, "silence transcribes to nothing")
	assert.True(t, detail.Filtered, "segments WERE inspected — there simply were none")
	assert.False(t, c.verboseUnsupported.Load(), "silence is not evidence the engine lacks verbose_json")

	silent = false
	srt, err := c.Transcribe(context.Background(), audio)
	require.NoError(t, err)
	assert.Contains(t, srt, "Hello world", "filtering must still be live for later chunks")

	for _, f := range rec.seen() {
		assert.Equal(t, "verbose_json", f, "a silent chunk must not trigger a wasted srt request")
	}
}

// 🔴 A transient fault must NEVER latch the fallback: one bad minute would
// otherwise disable hallucination filtering for the life of the process, with
// no signal that it happened.
func TestWhisperClient_TransientFailureDoesNotLatchTheFallback(t *testing.T) {
	rec := &formatRecorder{}
	var failing atomic.Bool
	failing.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(t, r)
		if failing.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(verboseJSONOneCue))
	}))
	defer server.Close()

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	audio := writeTempAudio(t)

	_, err := c.Transcribe(context.Background(), audio)
	require.Error(t, err, "a 5xx that outlives the retry budget must surface")
	assert.False(t, c.verboseUnsupported.Load(), "5xx is not evidence the engine lacks verbose_json")

	failing.Store(false)
	srt, err := c.Transcribe(context.Background(), audio)
	require.NoError(t, err)
	assert.Contains(t, srt, "Hello world")

	for _, f := range rec.seen() {
		assert.Equal(t, "verbose_json", f, "no request should ever have degraded to srt")
	}
}

// 🔴 The fallback issues a SECOND request. Billing the same audio twice would
// burn the 9R-11 run budget at double rate and trip the ceiling mid-film.
func TestWhisperClient_MetersAudioExactlyOnceAcrossTheFallback(t *testing.T) {
	rec := &formatRecorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rec.record(t, r) == "verbose_json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Write([]byte("1\n00:00:01,000 --> 00:00:02,000\nHi\n\n"))
	}))
	defer server.Close()

	f, err := os.CreateTemp(t.TempDir(), "meter-*.wav")
	require.NoError(t, err)
	byteRate := uint32(32000)
	writeWAVWithChunks(t, f, byteRate, byteRate*30, false) // 30 seconds

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	b := NewBudget(0)
	ctx := WithBudget(context.Background(), b)

	_, err = c.Transcribe(ctx, f.Name())
	require.NoError(t, err)

	snap := b.Snapshot()
	assert.Equal(t, 1, snap.ASRCalls, "two HTTP requests, ONE billable transcription")
	assert.InDelta(t, 30.0, snap.ASRSeconds, 0.01)
	assert.Equal(t, []string{"verbose_json", "srt"}, rec.seen())
}

func TestWhisperClient_TranscribeDetailed_ReportsWhatTheFilterRemoved(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"language":"english","duration":109.0,"segments":[
		  {"id":0,"start":10.0,"end":12.0,"text":"Real dialogue","avg_logprob":-0.25,"compression_ratio":1.4,"no_speech_prob":0.01},
		  {"id":1,"start":13.0,"end":15.0,"text":"More dialogue","avg_logprob":-0.3,"compression_ratio":1.3,"no_speech_prob":0.02},
		  {"id":2,"start":100.0,"end":103.0,"text":"Thanks for watching!","avg_logprob":-0.9,"compression_ratio":1.5,"no_speech_prob":0.55},
		  {"id":3,"start":103.0,"end":106.0,"text":"Like and subscribe.","avg_logprob":-1.2,"compression_ratio":1.6,"no_speech_prob":0.72},
		  {"id":4,"start":106.0,"end":109.0,"text":"See you next time.","avg_logprob":-0.95,"compression_ratio":1.4,"no_speech_prob":0.61}
		]}`))
	}))
	defer server.Close()

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	detail, err := c.TranscribeDetailed(context.Background(), writeTempAudio(t), "en")
	require.NoError(t, err)

	assert.True(t, detail.Filtered)
	assert.Equal(t, 5, detail.SegmentsIn)
	assert.Equal(t, 2, detail.SegmentsKept)
	assert.InDelta(t, 0.6, detail.DropRatio(), 0.001)
	assert.NotContains(t, detail.SRT, "subscribe")
	assert.Contains(t, detail.SRT, "Real dialogue")
	assert.Contains(t, detail.Unfiltered, "subscribe", "the unfiltered rendering is the recovery path")
	assert.Equal(t, 5, strings.Count(detail.Unfiltered, " --> "))
}

// When the engine cannot serve verbose_json there are no segments to judge, so
// the detail must say so rather than implying an empty filter pass.
func TestWhisperClient_TranscribeDetailed_FallbackReportsUnfiltered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(32<<20))
		if r.FormValue("response_format") == "verbose_json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Write([]byte("1\n00:00:01,000 --> 00:00:02,000\nPlain\n\n"))
	}))
	defer server.Close()

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	detail, err := c.TranscribeDetailed(context.Background(), writeTempAudio(t), "")
	require.NoError(t, err)

	assert.False(t, detail.Filtered)
	assert.Equal(t, detail.SRT, detail.Unfiltered)
	assert.Zero(t, detail.SegmentsIn)
	assert.Zero(t, detail.DropRatio())
}

func TestWhisperClient_SatisfiesDetailedTranscriber(t *testing.T) {
	var _ DetailedTranscriber = NewWhisperClient("k")
}

// AC #4: a chunk that collapses to nothing must not corrupt the merge — the
// remaining chunks keep contiguous numbering and correctly shifted timestamps.
func TestMergeSRTChunks_HandlesAFullyFilteredChunk(t *testing.T) {
	first := segmentsToSRT([]whisperSegment{speech(1, 2, "One"), speech(3, 4, "Two")})
	emptied := "" // an entire ten minutes of hallucinated credits
	third := segmentsToSRT([]whisperSegment{speech(5, 6, "Three")})

	merged := MergeSRTChunks([]string{first, emptied, third}, 600)

	assert.Contains(t, merged, "1\n00:00:01,000 --> 00:00:02,000\nOne")
	assert.Contains(t, merged, "2\n00:00:03,000 --> 00:00:04,000\nTwo")
	assert.Contains(t, merged, "3\n00:20:05,000 --> 00:20:06,000\nThree",
		"the third chunk keeps its 2x600s offset and continues the numbering")
	assert.NotContains(t, merged, "4\n")
}
