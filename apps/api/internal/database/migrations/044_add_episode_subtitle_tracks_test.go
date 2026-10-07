package migrations

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMigration044_AddsNullableSubtitleTracksToEpisodes(t *testing.T) {
	db := newFullyMigratedDB(t)
	now := time.Now().UTC()

	_, err := db.Exec(`INSERT INTO series (id, title, first_air_date, created_at, updated_at) VALUES ('s1', 'See', '2019-11-01', ?, ?)`, now, now)
	require.NoError(t, err)

	// A pre-044 shaped insert reads back NULL: "never read" — the verdict must
	// see unknown, not "no subtitles".
	_, err = db.Exec(`INSERT INTO episodes (id, series_id, season_number, episode_number, created_at, updated_at)
		VALUES ('legacy', 's1', 1, 1, ?, ?)`, now, now)
	require.NoError(t, err)
	var tracks, sig sql.NullString
	require.NoError(t, db.QueryRow(`SELECT subtitle_tracks, subtitle_tracks_file_sig FROM episodes WHERE id = 'legacy'`).Scan(&tracks, &sig))
	assert.False(t, tracks.Valid)
	assert.False(t, sig.Valid)

	_, err = db.Exec(`INSERT INTO episodes (id, series_id, season_number, episode_number, subtitle_tracks, subtitle_tracks_file_sig, created_at, updated_at)
		VALUES ('probed', 's1', 1, 2, '[]', '10:20', ?, ?)`, now, now)
	require.NoError(t, err)
	require.NoError(t, db.QueryRow(`SELECT subtitle_tracks, subtitle_tracks_file_sig FROM episodes WHERE id = 'probed'`).Scan(&tracks, &sig))
	assert.Equal(t, "[]", tracks.String)
	assert.Equal(t, "10:20", sig.String)
}

func TestMigration044_UpIsIdempotent(t *testing.T) {
	db := newFullyMigratedDB(t)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, (&addEpisodeSubtitleTracks{}).Up(tx))
	require.NoError(t, tx.Commit())
}

func TestMigration044_IsRegistered(t *testing.T) {
	var found bool
	for _, m := range GetAll() {
		if m.Version() == 44 {
			found = true
			assert.Equal(t, "add_episode_subtitle_tracks", m.Name())
		}
	}
	assert.True(t, found)
}
