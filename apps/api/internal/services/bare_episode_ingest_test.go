package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/parser"
	"github.com/vido/api/internal/repository"
)

// ─── disc-2026-10-bare-episode-filename-uses-folder-title ───────────────────

const ringsDir = "/media/tv/The Lord of the Rings - The Rings of Power"

func newBareEpisodeIngest(t *testing.T) (*MediaIngestService, *repository.SeriesRepository, *repository.EpisodeRepository) {
	t.Helper()
	db := setupTestDB(t)
	series := repository.NewSeriesRepository(db)
	episodes := repository.NewEpisodeRepository(db)
	return NewMediaIngestService(series, repository.NewSeasonRepository(db), episodes, nil), series, episodes
}

func TestIngest_BareEpisodeFileJoinsTheFoldersShow(t *testing.T) {
	ingest, series, episodes := newBareEpisodeIngest(t)
	ctx := context.Background()
	file := ringsDir + "/Season 1/S01E03.mkv"

	id, created, err := ingest.IngestEpisodeFile(ctx, file, "/media/tv", "", parser.NewTVParser().Parse("S01E03.mkv"))
	require.NoError(t, err)
	assert.True(t, created)

	show, err := series.FindByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "The Lord of the Rings - The Rings of Power", show.Title, "the folder names the show")
	ep, err := episodes.FindBySeriesSeasonEpisode(ctx, id, 1, 3)
	require.NoError(t, err)
	assert.Equal(t, file, ep.FilePath.String)
}

// The NAS's 力量之戒 S1 still has an episode-0 row holding S01E08.mkv from
// before failed parses stopped collapsing onto it. Once E08 parses, the file
// lives on episode 8 only.
func TestIngest_FileLeavesTheOldEpisodeZeroRow(t *testing.T) {
	ingest, series, episodes := newBareEpisodeIngest(t)
	ctx := context.Background()
	file := ringsDir + "/Season 1/S01E08.mkv"
	require.NoError(t, series.Create(ctx, &models.Series{ID: "rings", Title: "魔戒：力量之戒", FilePath: models.NewNullString(ringsDir)}))
	require.NoError(t, episodes.Create(ctx, &models.Episode{
		ID: "ep0", SeriesID: "rings", SeasonNumber: 1, EpisodeNumber: 0,
		FilePath: models.NewNullString(file), SubtitleStatus: models.SubtitleStatusNotSearched,
	}))
	other := ringsDir + "/Season 1/S01E09.mkv"
	require.NoError(t, episodes.Create(ctx, &models.Episode{
		ID: "ep9", SeriesID: "rings", SeasonNumber: 1, EpisodeNumber: 9,
		FilePath: models.NewNullString(other), SubtitleStatus: models.SubtitleStatusNotSearched,
	}))

	id, _, err := ingest.IngestEpisodeFile(ctx, file, "/media/tv", "", parser.NewTVParser().Parse("S01E08.mkv"))
	require.NoError(t, err)
	require.Equal(t, "rings", id)

	ep8, err := episodes.FindBySeriesSeasonEpisode(ctx, "rings", 1, 8)
	require.NoError(t, err)
	assert.Equal(t, file, ep8.FilePath.String)
	ep0, err := episodes.FindByID(ctx, "ep0")
	require.NoError(t, err, "the row stays")
	assert.False(t, ep0.FilePath.Valid, "one file, one episode")
	ep9, err := episodes.FindByID(ctx, "ep9")
	require.NoError(t, err)
	assert.Equal(t, other, ep9.FilePath.String, "a different file is not touched")
}

// Scanning the same file again keeps it on its own row.
func TestIngest_RescanKeepsTheFileOnItsRow(t *testing.T) {
	ingest, _, episodes := newBareEpisodeIngest(t)
	ctx := context.Background()
	file := ringsDir + "/Season 1/S01E03.mkv"
	parsed := parser.NewTVParser().Parse("S01E03.mkv")

	id, _, err := ingest.IngestEpisodeFile(ctx, file, "/media/tv", "", parsed)
	require.NoError(t, err)
	_, created, err := ingest.IngestEpisodeFile(ctx, file, "/media/tv", "", parsed)
	require.NoError(t, err)
	assert.False(t, created)

	ep, err := episodes.FindBySeriesSeasonEpisode(ctx, id, 1, 3)
	require.NoError(t, err)
	assert.Equal(t, file, ep.FilePath.String)
}
