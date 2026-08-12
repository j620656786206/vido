package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	_ "modernc.org/sqlite"
)

// newMigratedEpisodeDB applies the REAL migration chain rather than a
// hand-written CREATE TABLE. Rule 15 / bugfix-20-1: a hand-rolled test schema
// cannot catch a predicate that references a column the shipped schema does not
// have — which is exactly the class of bug this enumeration could hide (the
// episodes table has no is_removed column, unlike movies).
func newMigratedEpisodeDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

// seedGenerationEpisode inserts one episode row directly. episodeNumber must be
// unique per series — the shipped schema carries a UNIQUE index on
// (series_id, season_number, episode_number).
func seedGenerationEpisode(t *testing.T, db *sql.DB, id, seriesID string, episodeNumber int, filePath, subtitleLanguage string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO episodes (id, series_id, season_number, episode_number, title, file_path, subtitle_language, subtitle_status)
		VALUES (?, ?, 1, ?, ?, ?, ?, 'not_searched')`,
		id, seriesID, episodeNumber, fmt.Sprintf("Ep %d", episodeNumber),
		nullableText(filePath), nullableText(subtitleLanguage))
	require.NoError(t, err)
}

func nullableText(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func seedGenerationSeries(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO series (id, title, first_air_date) VALUES (?, ?, ?)`,
		id, "Show "+id, "2020-01-01")
	require.NoError(t, err)
}

// TestEpisodeRepository_FindMissingZhHantSubtitle covers the episode-grain
// enumeration sub-1-6 AC #11 added (Rule 24 lane ① — 9R-16 shipped movies-only).
// The predicate deliberately MIRRORS the movie one: "no zh-Hant on record AND a
// media file present", which is broader than a subtitle_status filter because
// an episode with a found ENGLISH subtitle still lacks zh-Hant.
func TestEpisodeRepository_FindMissingZhHantSubtitle(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewEpisodeRepository(db)
	seedGenerationSeries(t, db, "s1")

	seedGenerationEpisode(t, db, "needs-generation", "s1", 1, "/media/s1e1.mkv", "")
	seedGenerationEpisode(t, db, "has-english", "s1", 2, "/media/s1e2.mkv", "en")
	seedGenerationEpisode(t, db, "has-simplified", "s1", 3, "/media/s1e3.mkv", "zh-Hans")
	seedGenerationEpisode(t, db, "already-done", "s1", 4, "/media/s1e4.mkv", "zh-Hant")
	seedGenerationEpisode(t, db, "no-media-file", "s1", 5, "", "")
	seedGenerationEpisode(t, db, "empty-media-path", "s1", 6, "", "en")

	got, err := repo.FindMissingZhHantSubtitle(context.Background())
	require.NoError(t, err)

	ids := make([]string, 0, len(got))
	for _, e := range got {
		ids = append(ids, e.ID)
	}

	assert.ElementsMatch(t, []string{"needs-generation", "has-english", "has-simplified"}, ids)
	assert.NotContains(t, ids, "already-done",
		"a completed episode must self-exclude, which is what makes a re-scan free")
	assert.NotContains(t, ids, "no-media-file",
		"there is nothing to extract a track from without a media file")
	assert.NotContains(t, ids, "empty-media-path",
		"a NULL file_path is as unusable as a missing row")
}

// TestEpisodeRepository_FindMissingZhHantSubtitle_LoadsTheFieldsTheEnqueueNeeds —
// Rule 15 DB column sync: the enqueue builds a MediaRef from ID, and sub-1-6's
// MediaStore adapter resolves series_id as the D10 show key. A SELECT that
// dropped either would silently return the zero value.
func TestEpisodeRepository_FindMissingZhHantSubtitle_LoadsTheFieldsTheEnqueueNeeds(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewEpisodeRepository(db)
	seedGenerationSeries(t, db, "s1")
	seedGenerationEpisode(t, db, "ep-1", "s1", 1, "/media/s1e1.mkv", "")

	got, err := repo.FindMissingZhHantSubtitle(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)

	assert.Equal(t, "ep-1", got[0].ID)
	assert.Equal(t, "s1", got[0].SeriesID, "series_id is the D10 show key")
	assert.Equal(t, "/media/s1e1.mkv", got[0].FilePath.String)
	assert.Equal(t, models.SubtitleStatusNotSearched, got[0].SubtitleStatus)
}

// TestEpisodeRepository_FindMissingZhHantSubtitle_OrdersForTheShowGate — the
// ordering is load-bearing, not cosmetic: grouping a show's episodes together
// is what lets sub-1-5b's D10 latch warm ONE prompt prefix per show instead of
// re-warming it every time the queue interleaves two series.
func TestEpisodeRepository_FindMissingZhHantSubtitle_OrdersForTheShowGate(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewEpisodeRepository(db)
	seedGenerationSeries(t, db, "s-a")
	seedGenerationSeries(t, db, "s-b")

	// Interleaved insert order — the query must not preserve it.
	seedGenerationEpisode(t, db, "b2", "s-b", 2, "/media/b2.mkv", "")
	seedGenerationEpisode(t, db, "a2", "s-a", 2, "/media/a2.mkv", "")
	seedGenerationEpisode(t, db, "b1", "s-b", 1, "/media/b1.mkv", "")
	seedGenerationEpisode(t, db, "a1", "s-a", 1, "/media/a1.mkv", "")

	got, err := repo.FindMissingZhHantSubtitle(context.Background())
	require.NoError(t, err)

	ids := make([]string, 0, len(got))
	for _, e := range got {
		ids = append(ids, e.ID)
	}
	assert.Equal(t, []string{"a1", "a2", "b1", "b2"}, ids,
		"episodes come out grouped by series, in broadcast order")
}

func TestEpisodeRepository_FindMissingZhHantSubtitle_EmptyLibrary(t *testing.T) {
	repo := NewEpisodeRepository(newMigratedEpisodeDB(t))

	got, err := repo.FindMissingZhHantSubtitle(context.Background())
	require.NoError(t, err)
	assert.Empty(t, got, "an empty library is not an error")
}

// TestEpisodeRepository_CountMissingZhHantSubtitle — the count twin (sub-5-1
// AC #7) must agree with the enumeration by construction: same shared
// predicate, so the three-段 assertion (mixed set → writeback → shrink)
// mirrors the movie repo's pattern.
func TestEpisodeRepository_CountMissingZhHantSubtitle(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewEpisodeRepository(db)
	seedGenerationSeries(t, db, "s1")

	seedGenerationEpisode(t, db, "needs-generation", "s1", 1, "/media/s1e1.mkv", "")
	seedGenerationEpisode(t, db, "has-english", "s1", 2, "/media/s1e2.mkv", "en")
	seedGenerationEpisode(t, db, "already-done", "s1", 3, "/media/s1e3.mkv", "zh-Hant")
	seedGenerationEpisode(t, db, "no-media-file", "s1", 4, "", "")

	count, err := repo.CountMissingZhHantSubtitle(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Count and enumeration share one predicate — they can never disagree.
	found, err := repo.FindMissingZhHantSubtitle(context.Background())
	require.NoError(t, err)
	assert.Len(t, found, count)

	// The generation-success writeback self-excludes the episode.
	require.NoError(t, repo.UpdateEpisodeSubtitleStatus(context.Background(),
		"needs-generation", models.SubtitleStatusFound, "/media/s1e1.zh-Hant.srt", "zh-Hant"))
	count, err = repo.CountMissingZhHantSubtitle(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestEpisodeRepository_CountMissingZhHantSubtitle_EmptyLibrary(t *testing.T) {
	repo := NewEpisodeRepository(newMigratedEpisodeDB(t))
	count, err := repo.CountMissingZhHantSubtitle(context.Background())
	require.NoError(t, err)
	assert.Zero(t, count)
}
