package services

// Story disc-2026-07-transcription-active-jobs: a single-episode/movie
// transcription kept no discoverable record beyond a single-key dedup map, so
// closing the progress modal made the job invisible everywhere, including the
// Activity page. These tests pin the two new primitives that close that gap:
// resolveActivityTitle (fail-soft title lookup) and ActivityProgress (the
// batchJobSource shape every other Activity source already implements).

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

func newActivityTestService() *TranscriptionService {
	return NewTranscriptionService(nil, nil, nil, nil)
}

func TestResolveActivityTitle_Episode(t *testing.T) {
	svc := newActivityTestService()
	svc.SetEpisodeSubtitleStateReader(&fakeEpisodeStateReader{
		episode: &models.Episode{
			ID:            "ep-1",
			SeriesID:      "series-1",
			SeasonNumber:  3,
			EpisodeNumber: 1,
		},
	})
	svc.SetSeriesMetadataReader(&metadataSeriesReader{
		series: &models.Series{ID: "series-1", Title: "龍族前傳"},
	})

	got := svc.resolveActivityTitle(context.Background(), models.SubtitleRunMediaEpisode, "ep-1")

	assert.Equal(t, "龍族前傳 S03E01", got)
}

func TestResolveActivityTitle_Movie(t *testing.T) {
	svc := newActivityTestService()
	svc.SetSubtitleStateReader(&fakeStateReader{
		movie: &models.Movie{ID: "movie-1", Title: "阿凡達：火與燼"},
	})

	got := svc.resolveActivityTitle(context.Background(), models.SubtitleRunMediaMovie, "movie-1")

	assert.Equal(t, "阿凡達：火與燼", got)
}

func TestResolveActivityTitle_EpisodeLookupFails_FallsBackToMediaID(t *testing.T) {
	svc := newActivityTestService()
	svc.SetEpisodeSubtitleStateReader(&fakeEpisodeStateReader{err: errors.New("db down")})

	got := svc.resolveActivityTitle(context.Background(), models.SubtitleRunMediaEpisode, "ep-missing")

	// A display-string lookup failure must never fabricate a title, and must
	// never block or fail the transcription itself over it.
	assert.Equal(t, "ep-missing", got)
}

func TestResolveActivityTitle_EpisodeFoundButSeriesLookupFails_FallsBackToMediaID(t *testing.T) {
	svc := newActivityTestService()
	svc.SetEpisodeSubtitleStateReader(&fakeEpisodeStateReader{
		episode: &models.Episode{ID: "ep-1", SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 1},
	})
	svc.SetSeriesMetadataReader(&metadataSeriesReader{err: errors.New("db down")})

	got := svc.resolveActivityTitle(context.Background(), models.SubtitleRunMediaEpisode, "ep-1")

	assert.Equal(t, "ep-1", got)
}

func TestResolveActivityTitle_MovieLookupFails_FallsBackToMediaID(t *testing.T) {
	svc := newActivityTestService()
	svc.SetSubtitleStateReader(&fakeStateReader{err: errors.New("db down")})

	got := svc.resolveActivityTitle(context.Background(), models.SubtitleRunMediaMovie, "movie-missing")

	assert.Equal(t, "movie-missing", got)
}

func TestResolveActivityTitle_NoReaderWired_FallsBackToMediaID(t *testing.T) {
	svc := newActivityTestService()

	got := svc.resolveActivityTitle(context.Background(), models.SubtitleRunMediaEpisode, "ep-1")

	assert.Equal(t, "ep-1", got)
}

func TestActivityProgress_NoJobs(t *testing.T) {
	svc := newActivityTestService()

	active, pct, cur, total, detail := svc.ActivityProgress()

	assert.False(t, active)
	assert.Equal(t, 0, pct)
	assert.Equal(t, 0, cur)
	assert.Equal(t, 0, total)
	assert.Equal(t, "", detail)
}

func TestActivityProgress_OneJob(t *testing.T) {
	svc := newActivityTestService()
	_, err := svc.acquireJob("media-1", "龍族前傳 S03E01", true)
	require.NoError(t, err)

	active, pct, cur, total, detail := svc.ActivityProgress()

	assert.True(t, active)
	// This service tracks discrete stages, not a fractional/bounded count —
	// fabricating a percent or a total would be dishonest, so both stay 0.
	assert.Equal(t, 0, pct)
	assert.Equal(t, 0, total)
	assert.Equal(t, 1, cur)
	assert.Equal(t, "龍族前傳 S03E01", detail)
}

func TestActivityProgress_MultipleJobs(t *testing.T) {
	svc := newActivityTestService()
	_, err := svc.acquireJob("media-1", "龍族前傳 S03E01", true)
	require.NoError(t, err)
	_, err = svc.acquireJob("media-2", "阿凡達：火與燼", true)
	require.NoError(t, err)

	active, pct, cur, total, detail := svc.ActivityProgress()

	assert.True(t, active)
	assert.Equal(t, 0, pct)
	assert.Equal(t, 0, total)
	assert.Equal(t, 2, cur)
	// Which of the two concurrent jobs' titles surfaces is unspecified (map
	// iteration order) — any currently-running job's title is an honest
	// answer to "something is generating right now." Assert membership, not
	// a specific one.
	assert.Contains(t, []string{"龍族前傳 S03E01", "阿凡達：火與燼"}, detail)
}

func TestActivityProgress_JobCompletion_RemovesFromCount(t *testing.T) {
	svc := newActivityTestService()
	jobID, err := svc.acquireJob("media-1", "龍族前傳 S03E01", true)
	require.NoError(t, err)
	require.NotEmpty(t, jobID)

	delete(svc.inProgress, "media-1")

	active, _, cur, _, detail := svc.ActivityProgress()

	assert.False(t, active)
	assert.Equal(t, 0, cur)
	assert.Equal(t, "", detail)
}

// The core reason acquireJob grew a solo flag: RunTranscription (batch/
// pipeline path) shares the SAME single-flight map as StartTranscription
// (solo path) for dedup, but a batch item already has its own Activity row
// (generation_batch) — counting it again here would double-count one real
// job as two rows on the Activity page.
func TestActivityProgress_NonSoloJob_DoesNotCount(t *testing.T) {
	svc := newActivityTestService()
	_, err := svc.acquireJob("media-1", "", false) // RunTranscription's shape
	require.NoError(t, err)

	active, _, cur, _, detail := svc.ActivityProgress()

	assert.False(t, active)
	assert.Equal(t, 0, cur)
	assert.Equal(t, "", detail)
}

func TestActivityProgress_MixOfSoloAndBatchJobs_CountsSoloOnly(t *testing.T) {
	svc := newActivityTestService()
	_, err := svc.acquireJob("media-1", "龍族前傳 S03E01", true)
	require.NoError(t, err)
	_, err = svc.acquireJob("media-2", "", false)
	require.NoError(t, err)

	active, _, cur, _, detail := svc.ActivityProgress()

	assert.True(t, active)
	assert.Equal(t, 1, cur)
	assert.Equal(t, "龍族前傳 S03E01", detail)
}

// Compile-time interface verification — mirrors the pattern used across this
// codebase (e.g. GeminiProvider/ClaudeProvider against ai.Provider).
func TestTranscriptionService_ImplementsBatchJobSource(t *testing.T) {
	var _ interface {
		ActivityProgress() (active bool, percentDone, current, total int, currentItem string)
	} = (*TranscriptionService)(nil)
}
