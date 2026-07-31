package subtitle

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/services"
)

// Live-file guard for the Chinese-first routing fix (2026-07-31).
//
// The unit tests fake the extractor; this one drives the REAL ffprobe + ffmpeg
// against a real multi-Chinese-track file, which is how the defect was found in
// the first place (Apple TV+ files carry Simplified / Traditional / Cantonese
// tracks with identical cue counts, all tagged plain `chi`). Opt-in via
// VIDO_LIVE_MEDIA so CI and other machines skip it:
//
//	VIDO_LIVE_MEDIA=/path/to/file.mkv go test ./internal/subtitle/ -run LiveFile -v
func TestSelectAndRoute_LiveFile_PrefersEmbeddedTraditional(t *testing.T) {
	mediaPath := os.Getenv("VIDO_LIVE_MEDIA")
	if mediaPath == "" {
		t.Skip("VIDO_LIVE_MEDIA not set — skipping live-file routing case")
	}
	if _, err := os.Stat(mediaPath); err != nil {
		t.Skipf("VIDO_LIVE_MEDIA not reachable (%v) — skipping", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	prober := services.NewFFprobeService(1, 10*time.Second, logger)
	extractor := NewExtractor(10*time.Minute, logger)
	r := NewRouter(prober, extractor, logger)

	got, err := r.SelectAndRoute(context.Background(), mediaPath, t.TempDir())
	require.NoError(t, err)

	t.Logf("verdict=%s variant=%s reason=%s", got.Kind, got.DetectedVariant, got.Reason)
	require.NotNil(t, got.Track, "a file with embedded Chinese must yield a track")
	t.Logf("chosen stream=%d tag=%s codec=%s cues=%d",
		got.Track.StreamIndex, got.Track.Language, got.Track.Codec, len(got.Track.Blocks))

	// The whole point of the fix: an already-Traditional embedded track ships
	// as-is — no OpenCC round-trip and, above all, no LLM translation bill.
	assert.Equal(t, RouteDeliverDirect, got.Kind)
	assert.Equal(t, LangTraditional, got.DetectedVariant)
	assert.NotEmpty(t, got.Track.Blocks)
}
