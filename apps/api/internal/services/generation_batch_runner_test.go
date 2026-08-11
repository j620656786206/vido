package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

// fakeTranscriptionSeam records the RunTranscription call so the test can
// apply the forwarded options and assert their EFFECT (translate + media-type
// dispatch) rather than comparing opaque function values.
type fakeTranscriptionSeam struct {
	available bool
	err       error

	gotMediaID  string
	gotFilePath string
	gotMediaDir string
	gotOpts     []TranscriptionOption
}

func (f *fakeTranscriptionSeam) IsAvailable() bool { return f.available }

func (f *fakeTranscriptionSeam) RunTranscription(_ context.Context, mediaID, filePath, mediaDir string, opts ...TranscriptionOption) error {
	f.gotMediaID = mediaID
	f.gotFilePath = filePath
	f.gotMediaDir = mediaDir
	f.gotOpts = opts
	return f.err
}

// sub-4-2 AC #5: the legacy Route C runner must forward BOTH WithTranslation
// (pre-sub-4-2 behavior preserved) and WithMediaType — a missing media type
// silently defaults to movie and writes an episode's status onto the movies
// table (transcription_service.go newTranscriptionConfig default).
func TestRouteCGenerationRunner_ForwardsMediaTypeAndTranslation(t *testing.T) {
	seam := &fakeTranscriptionSeam{available: true}
	r := NewRouteCGenerationRunner(seam)

	err := r.ExecuteGeneration(context.Background(), "ep-1", models.SubtitleRunMediaEpisode, "/tv/s01e01.mkv", "/tv")
	require.NoError(t, err)
	assert.Equal(t, "ep-1", seam.gotMediaID)
	assert.Equal(t, "/tv/s01e01.mkv", seam.gotFilePath)
	assert.Equal(t, "/tv", seam.gotMediaDir)

	cfg := newTranscriptionConfig(seam.gotOpts)
	assert.True(t, cfg.translate, "WithTranslation must be preserved (pre-sub-4-2 behavior)")
	assert.Equal(t, models.SubtitleRunMediaEpisode, cfg.mediaType,
		"WithMediaType must reach the service — the config default is movie")
}

func TestRouteCGenerationRunner_MovieStaysMovie(t *testing.T) {
	seam := &fakeTranscriptionSeam{available: true}
	r := NewRouteCGenerationRunner(seam)

	require.NoError(t, r.ExecuteGeneration(context.Background(), "m-1", models.SubtitleRunMediaMovie, "/m/a.mkv", "/m"))
	cfg := newTranscriptionConfig(seam.gotOpts)
	assert.Equal(t, models.SubtitleRunMediaMovie, cfg.mediaType)
	assert.True(t, cfg.translate)
}

// AC #6: the runner must preserve the error chain — the orchestrator
// classifies errors.Is(err, ai.ErrBudgetExceeded) as a budget_ceiling PAUSE.
func TestRouteCGenerationRunner_PreservesBudgetExceededChain(t *testing.T) {
	seam := &fakeTranscriptionSeam{available: true, err: errors.Join(errors.New("asr leg"), ai.ErrBudgetExceeded)}
	r := NewRouteCGenerationRunner(seam)

	err := r.ExecuteGeneration(context.Background(), "m-1", models.SubtitleRunMediaMovie, "/m/a.mkv", "/m")
	assert.ErrorIs(t, err, ai.ErrBudgetExceeded)
}

func TestRouteCGenerationRunner_IsAvailable(t *testing.T) {
	assert.False(t, RouteCGenerationRunner{}.IsAvailable(), "nil seam must not panic")
	assert.False(t, NewRouteCGenerationRunner(&fakeTranscriptionSeam{available: false}).IsAvailable())
	assert.True(t, NewRouteCGenerationRunner(&fakeTranscriptionSeam{available: true}).IsAvailable())
}
