package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

type pinQuoter struct {
	got         TranscriptionEstimateTarget
	usd         float64
	hadDeadline bool
}

func (q *pinQuoter) Estimate(ctx context.Context, target TranscriptionEstimateTarget) TranscriptionEstimate {
	q.got = target
	_, q.hadDeadline = ctx.Deadline()
	return TranscriptionEstimate{EstimatedUSD: q.usd}
}

type stubRunMovies struct {
	movie *models.Movie
	err   error
}

func (s stubRunMovies) FindByID(context.Context, string) (*models.Movie, error) {
	return s.movie, s.err
}

type stubRunEpisodes struct {
	episode *models.Episode
	err     error
}

func (s stubRunEpisodes) FindByID(context.Context, string) (*models.Episode, error) {
	return s.episode, s.err
}

// The run-start quote is built exactly as the estimate handler builds it, so
// the ledger's figure is the dialog's figure (AC #2).
func TestTranscriptionRunEstimator_MovieTargetMirrorsTheHandler(t *testing.T) {
	movie := &models.Movie{ID: "mv1"}
	movie.FilePath = models.NewNullString("/media/m.mkv")
	movie.DurationSeconds = models.NewNullInt64(5400)
	movie.Runtime = models.NewNullInt64(91)
	movie.SubtitleTracks = models.NewNullString(`[{"index":2}]`)
	q := &pinQuoter{usd: 0.73}

	usd, err := NewTranscriptionRunEstimator(q, stubRunMovies{movie: movie}, nil).
		EstimateRunUSD(context.Background(), models.SubtitleRunMediaMovie, "mv1")

	require.NoError(t, err)
	assert.InDelta(t, 0.73, usd, 1e-9)
	assert.True(t, q.hadDeadline, "a cold probe must not hold up the run start")
	assert.Equal(t, TranscriptionEstimateTarget{
		MediaID: "mv1", MediaType: models.SubtitleRunMediaMovie, FilePath: "/media/m.mkv",
		DurationSeconds: movie.DurationSeconds, Runtime: movie.Runtime, SubtitleTracksJSON: `[{"index":2}]`,
	}, q.got)
}

func TestTranscriptionRunEstimator_EpisodeTargetMirrorsTheHandler(t *testing.T) {
	ep := &models.Episode{ID: "ep1"}
	ep.FilePath = models.NewNullString("/media/s01e01.mkv")
	ep.Runtime = models.NewNullInt64(44)
	q := &pinQuoter{usd: 0.2}

	usd, err := NewTranscriptionRunEstimator(q, nil, stubRunEpisodes{episode: ep}).
		EstimateRunUSD(context.Background(), models.SubtitleRunMediaEpisode, "ep1")

	require.NoError(t, err)
	assert.InDelta(t, 0.2, usd, 1e-9)
	assert.Equal(t, "/media/s01e01.mkv", q.got.FilePath)
	assert.Equal(t, models.SubtitleRunMediaEpisode, q.got.MediaType)
	assert.Equal(t, "", q.got.SubtitleTracksJSON, "the episode handler passes no track list either")
}

// Every "no figure" case is an error, which the ledger writers turn into NULL.
func TestTranscriptionRunEstimator_NoFigureIsAnError(t *testing.T) {
	q := &pinQuoter{}
	noPath := &models.Movie{ID: "mv1"}
	cases := map[string]*TranscriptionRunEstimator{
		"series":        NewTranscriptionRunEstimator(q, nil, nil),
		"lookup failed": NewTranscriptionRunEstimator(q, stubRunMovies{err: errors.New("db locked")}, nil),
		"not found":     NewTranscriptionRunEstimator(q, stubRunMovies{}, nil),
		"no file path":  NewTranscriptionRunEstimator(q, stubRunMovies{movie: noPath}, nil),
	}
	for name, est := range cases {
		mediaType := models.SubtitleRunMediaMovie
		if name == "series" {
			mediaType = models.SubtitleRunMediaSeries
		}
		_, err := est.EstimateRunUSD(context.Background(), mediaType, "mv1")
		assert.Error(t, err, name)
	}
	var nilEst *TranscriptionRunEstimator
	_, err := nilEst.EstimateRunUSD(context.Background(), models.SubtitleRunMediaMovie, "mv1")
	assert.Error(t, err)
}
