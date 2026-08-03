package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// These tests close `backlog-subtitle-status-writer-search-columns`, filed by
// sub-1-5b at discovery time and owned by sub-1-6 (which owns the MediaStore
// adapter that would otherwise wire straight onto the search-path writer).
//
// sub-1-5b AC #6.3: a GENERATED subtitle must leave `subtitle_search_score` AND
// `subtitle_last_searched` unset — extract-and-translate is not a search, and
// stamping the search columns would make a generated sidecar indistinguishable
// from a provider hit in the library UI. Score was already satisfied for free
// (`Score: 0` → NULL), but both `UpdateSubtitleStatus` writers stamp
// `subtitle_last_searched = now` unconditionally.

func searchColumns(t *testing.T, db *sql.DB, table, id string) (score sql.NullFloat64, lastSearched sql.NullTime) {
	t.Helper()
	require.NoError(t, db.QueryRow(
		`SELECT subtitle_search_score, subtitle_last_searched FROM `+table+` WHERE id = ?`, id,
	).Scan(&score, &lastSearched))
	return score, lastSearched
}

func TestMovieRepository_UpdateSubtitleGenerationStatus_LeavesSearchColumnsUnset(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewMovieRepository(db)
	ctx := context.Background()

	_, err := db.Exec(`INSERT INTO movies (id, title, release_date, file_path) VALUES ('m1', 'Movie', '2020-01-01', '/media/m1.mkv')`)
	require.NoError(t, err)

	// A prior SEARCH stamped the columns — the generation write must not
	// disturb them, and must not add its own stamp.
	require.NoError(t, repo.UpdateSubtitleStatus(ctx, "m1", models.SubtitleStatusFound, "/media/m1.en.srt", "en", 87.5))
	scoreBefore, searchedBefore := searchColumns(t, db, "movies", "m1")
	require.True(t, scoreBefore.Valid, "sanity: the search path DOES stamp a score")
	require.True(t, searchedBefore.Valid, "sanity: the search path DOES stamp subtitle_last_searched")

	require.NoError(t, repo.UpdateSubtitleGenerationStatus(ctx, "m1",
		models.SubtitleStatusFound, "/media/m1.zh-Hant.srt", "zh-Hant"))

	var status, path, language string
	require.NoError(t, db.QueryRow(
		`SELECT subtitle_status, subtitle_path, subtitle_language FROM movies WHERE id = 'm1'`,
	).Scan(&status, &path, &language))
	assert.Equal(t, string(models.SubtitleStatusFound), status)
	assert.Equal(t, "/media/m1.zh-Hant.srt", path)
	assert.Equal(t, "zh-Hant", language)

	scoreAfter, searchedAfter := searchColumns(t, db, "movies", "m1")
	assert.Equal(t, scoreBefore, scoreAfter, "generation must not touch subtitle_search_score")
	assert.Equal(t, searchedBefore, searchedAfter, "generation must not re-stamp subtitle_last_searched")
}

func TestMovieRepository_UpdateSubtitleGenerationStatus_NeverStampsOnAFreshRow(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewMovieRepository(db)

	_, err := db.Exec(`INSERT INTO movies (id, title, release_date, file_path) VALUES ('m1', 'Movie', '2020-01-01', '/media/m1.mkv')`)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateSubtitleGenerationStatus(context.Background(), "m1",
		models.SubtitleStatusExtracting, "", ""))

	score, lastSearched := searchColumns(t, db, "movies", "m1")
	assert.False(t, score.Valid, "a generated subtitle has no search score (AC #6.3)")
	assert.False(t, lastSearched.Valid,
		"extract-and-translate is not a search — the column must stay NULL")
}

func TestSeriesRepository_UpdateSubtitleGenerationStatus_LeavesSearchColumnsUnset(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	repo := NewSeriesRepository(db)
	ctx := context.Background()
	seedGenerationSeries(t, db, "s1")

	require.NoError(t, repo.UpdateSubtitleStatus(ctx, "s1", models.SubtitleStatusFound, "/media/s1.en.srt", "en", 91))
	scoreBefore, searchedBefore := searchColumns(t, db, "series", "s1")
	require.True(t, searchedBefore.Valid, "sanity: the search path DOES stamp subtitle_last_searched")

	require.NoError(t, repo.UpdateSubtitleGenerationStatus(ctx, "s1",
		models.SubtitleStatusFound, "/media/s1.zh-Hant.srt", "zh-Hant"))

	var language string
	require.NoError(t, db.QueryRow(`SELECT subtitle_language FROM series WHERE id = 's1'`).Scan(&language))
	assert.Equal(t, "zh-Hant", language)

	scoreAfter, searchedAfter := searchColumns(t, db, "series", "s1")
	assert.Equal(t, scoreBefore, scoreAfter)
	assert.Equal(t, searchedBefore, searchedAfter)
}

// TestUpdateSubtitleGenerationStatus_UnknownIDIsAnError — a silent zero-rows
// update would let the pipeline believe it had written a status it never wrote.
func TestUpdateSubtitleGenerationStatus_UnknownIDIsAnError(t *testing.T) {
	db := newMigratedEpisodeDB(t)

	assert.Error(t, NewMovieRepository(db).UpdateSubtitleGenerationStatus(
		context.Background(), "nope", models.SubtitleStatusFound, "", ""))
	assert.Error(t, NewSeriesRepository(db).UpdateSubtitleGenerationStatus(
		context.Background(), "nope", models.SubtitleStatusFound, "", ""))
}

// TestUpdateSubtitleStatus_SearchPathIsUnchanged — the Epic-8 search engine
// still needs the search columns. The new method is ADDITIVE; regressing the
// old one would silently blank the library UI's "last searched" data.
func TestUpdateSubtitleStatus_SearchPathIsUnchanged(t *testing.T) {
	db := newMigratedEpisodeDB(t)
	ctx := context.Background()

	_, err := db.Exec(`INSERT INTO movies (id, title, release_date, file_path) VALUES ('m1', 'Movie', '2020-01-01', '/media/m1.mkv')`)
	require.NoError(t, err)
	require.NoError(t, NewMovieRepository(db).UpdateSubtitleStatus(ctx, "m1",
		models.SubtitleStatusFound, "/media/m1.srt", "zh-Hant", 88.25))

	score, lastSearched := searchColumns(t, db, "movies", "m1")
	assert.InDelta(t, 88.25, score.Float64, 0.001)
	assert.True(t, lastSearched.Valid)
}
