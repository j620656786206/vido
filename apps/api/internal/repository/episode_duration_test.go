package repository

// sub-6-10a AC #1/#5(a) — duration_seconds must survive a real round trip.
//
// The failure class this guards is bugfix-20-1: a column that exists in the
// table and in the struct, but is missing from the SELECT list or the scan, so
// every read returns the zero value and the feature is silently dead. A test
// that only checked the migration would not catch it — this one writes through
// the repository and reads back through the repository.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

func TestEpisodeRepository_DurationSecondsRoundTrips(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	episode := &models.Episode{
		ID:              "ep-duration",
		SeriesID:        "series-1",
		SeasonNumber:    1,
		EpisodeNumber:   1,
		DurationSeconds: models.NewNullInt64(3300),
	}
	require.NoError(t, repo.Create(ctx, episode))

	found, err := repo.FindByID(ctx, "ep-duration")
	require.NoError(t, err)
	require.True(t, found.DurationSeconds.Valid, "written on Create, so it must read back — not a zero value")
	assert.EqualValues(t, 3300, found.DurationSeconds.Int64)
}

func TestEpisodeRepository_UnmeasuredEpisodeReadsNullNotZero(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Episode{
		ID: "ep-unmeasured", SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 2,
	}))

	found, err := repo.FindByID(ctx, "ep-unmeasured")
	require.NoError(t, err)
	assert.False(t, found.DurationSeconds.Valid,
		"never measured is NULL — a 0 here would price the episode as a zero-length file, i.e. free")
}

func TestEpisodeRepository_UpdateDurationSeconds(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Episode{
		ID: "ep-1", SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 1,
		Title: models.NewNullString("Innie"),
	}))

	require.NoError(t, repo.UpdateDurationSeconds(ctx, "ep-1", 3300))

	found, err := repo.FindByID(ctx, "ep-1")
	require.NoError(t, err)
	require.True(t, found.DurationSeconds.Valid)
	assert.EqualValues(t, 3300, found.DurationSeconds.Int64)
	assert.Equal(t, "Innie", found.Title.String,
		"a narrow single-column write must not disturb the rest of the row")
}

func TestEpisodeRepository_UpdateDurationSeconds_RejectsNonPositive(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Episode{
		ID: "ep-1", SeriesID: "series-1", SeasonNumber: 1, EpisodeNumber: 1,
	}))

	// 0 is ffprobe saying "no duration header". Storing it would convert
	// "unknown" into "zero-length", which the estimator prices as free.
	require.Error(t, repo.UpdateDurationSeconds(ctx, "ep-1", 0))
	require.Error(t, repo.UpdateDurationSeconds(ctx, "ep-1", -5))

	found, err := repo.FindByID(ctx, "ep-1")
	require.NoError(t, err)
	assert.False(t, found.DurationSeconds.Valid, "the rejected write must leave the column NULL")
}

func TestEpisodeRepository_UpdateDurationSeconds_UnknownEpisode(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	err := NewEpisodeRepository(db).UpdateDurationSeconds(context.Background(), "nope", 100)
	require.ErrorIs(t, err, ErrEpisodeNotFound)
}

func TestMovieRepository_DurationSecondsRoundTrips(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	movie := &models.Movie{
		ID:              "movie-duration",
		Title:           "Dune",
		ReleaseDate:     "2021-10-22",
		DurationSeconds: models.NewNullInt64(9960),
	}
	require.NoError(t, repo.Create(ctx, movie))

	found, err := repo.FindByID(ctx, "movie-duration")
	require.NoError(t, err)
	require.True(t, found.DurationSeconds.Valid)
	assert.EqualValues(t, 9960, found.DurationSeconds.Int64)
}
