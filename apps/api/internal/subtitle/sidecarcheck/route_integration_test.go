package sidecarcheck

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

// The whole route on a real file (-b2 AC #7): a film with no subtitle track
// and its English subtitle beside it. The router must choose translation from
// that file — no speech recognition — and leave the file byte for byte.
func TestRoute_ARealFilmTranslatesItsMatchingSidecar(t *testing.T) {
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not installed", bin)
		}
	}
	dir := t.TempDir()
	raw := filepath.Join(dir, "audio.pcm")
	require.NoError(t, os.WriteFile(raw, pcmFor(window{StartMS: 0, LenMS: 520_000}, timeline(fixtureCues(100))), 0o644))
	media := filepath.Join(dir, "Film.mkv")
	out, err := exec.Command("ffmpeg", "-loglevel", "error", "-f", "s16le", "-ar", "16000", "-ac", "1", "-i", raw,
		"-c:a", "aac", "-metadata:s:a:0", "language=eng", media).CombinedOutput()
	require.NoError(t, err, string(out))
	require.NoError(t, os.Remove(raw))

	fx := &finderFixture{dir: dir, media: media}
	sidecar := fx.write(t, "Film.en.srt", 0)
	before, err := os.ReadFile(sidecar)
	require.NoError(t, err)

	prober := services.NewFFprobeService(1, time.Minute, nil)
	finder := NewFinder(NewChecker(prober, nil, nil), services.NewAudioExtractorService(1, time.Minute, nil), nil)
	router := subtitle.NewRouter(prober, subtitle.NewExtractor(time.Minute, nil), nil)
	router.SetSidecarFinder(finder)

	got, err := router.SelectAndRoute(context.Background(), media, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, subtitle.RouteTranslate, got.Kind)
	require.NotNil(t, got.Track)
	assert.Equal(t, sidecar, got.Track.Path)
	assert.Len(t, got.Track.Blocks, 100)

	after, err := os.ReadFile(sidecar)
	require.NoError(t, err)
	assert.Equal(t, before, after, "the user's English subtitle is not touched")

	// The same film beside a subtitle for another cut: speech recognition, as before.
	require.NoError(t, os.Remove(sidecar))
	fx.write(t, "Film.en.srt", 40_000)
	got, err = router.SelectAndRoute(context.Background(), media, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, subtitle.RouteNoTextSource, got.Kind)
}
