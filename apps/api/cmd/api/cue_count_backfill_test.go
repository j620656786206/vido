package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/subtitle"
)

// disc-2026-11-backfill-subtitle-cue-counts AC #6, end to end with the REAL
// selection rule and the real route adapter: 《天蠍行動》 S01E02's stored
// English track has no count → the sweep re-reads the same file → 1,609 →
// the batch list prices it at about US$0.72 (was the runtime fallback, ~$0.35).

type scorpionRepo struct {
	ep models.Episode
}

func (r *scorpionRepo) FindWithFiles(context.Context) ([]models.Episode, error) {
	return []models.Episode{r.ep}, nil
}
func (r *scorpionRepo) UpdateSubtitleTracks(_ context.Context, _ string, tracks, sig string) error {
	r.ep.SubtitleTracks = models.NewNullString(tracks)
	r.ep.SubtitleTracksFileSig = models.NewNullString(sig)
	return nil
}
func (r *scorpionRepo) UpdateDurationSeconds(context.Context, string, int64) error { return nil }
func (r *scorpionRepo) FindMissingZhHantSubtitle(context.Context) ([]models.Episode, error) {
	return []models.Episode{r.ep}, nil
}

type scorpionProber struct{ calls int }

func (p *scorpionProber) IsAvailable() bool { return true }
func (p *scorpionProber) Probe(context.Context, string) (*services.MediaTechInfo, error) {
	p.calls++
	return &services.MediaTechInfo{SubtitleTracks: []services.SubtitleTrack{
		{Language: "eng", Format: "subrip", StreamIndex: 3, CueCount: 1609},
	}}, nil
}

func TestCueCountBackfill_ScorpionS01E02BatchQuote(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Scorpion.S01E02.mkv")
	require.NoError(t, os.WriteFile(path, []byte("video"), 0o644))
	info, err := os.Stat(path)
	require.NoError(t, err)

	repo := &scorpionRepo{ep: models.Episode{
		ID: "scorpion-s01e02", SeasonNumber: 1, EpisodeNumber: 2,
		FilePath:              models.NewNullString(path),
		DurationSeconds:       models.NewNullInt64(2464),
		SubtitleTracks:        models.NewNullString(`[{"language":"eng","format":"subrip","external":false,"stream_index":3}]`),
		SubtitleTracksFileSig: models.NewNullString(services.FileSignature(info)),
	}}
	quote := func() float64 {
		svc := services.NewGenerationCandidateService(nil, repo, nil, routePredictorAdapter{}, false, 0, nil)
		res, err := svc.Analyze(context.Background(), nil)
		require.NoError(t, err)
		require.Len(t, res.Candidates, 1)
		return res.Candidates[0].EstimatedUSD
	}
	assert.Equal(t, 0.35, quote(), "before: no stored count → 41 min × $0.0085")

	prober := &scorpionProber{}
	sweep := services.NewEpisodeSubtitleTracksService(repo, prober, nil, nil)
	sweep.SetCueCountRule(subtitle.CueCountRule{})
	res := sweep.Run(context.Background())
	assert.Equal(t, 1, res.CuesFilled)
	assert.Contains(t, repo.ep.SubtitleTracks.String, `"cue_count":1609`)

	assert.Equal(t, 0.72, quote(), "after: 1,609 × $0.45 per 1,000")
	assert.Zero(t, sweep.Run(context.Background()).Probed, "read once")
	assert.Equal(t, 1, prober.calls)
}
