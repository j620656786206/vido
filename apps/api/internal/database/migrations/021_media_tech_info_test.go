package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMediaTechInfo_Up(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create prerequisite tables (simplified)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS movies (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			release_date TEXT NOT NULL DEFAULT '',
			genres TEXT NOT NULL DEFAULT '[]',
			parse_status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS series (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			first_air_date TEXT NOT NULL DEFAULT '',
			genres TEXT NOT NULL DEFAULT '[]',
			parse_status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	// Insert test data before migration
	_, err = db.Exec(`
		INSERT INTO movies (id, title) VALUES ('movie-1', 'Test Movie');
		INSERT INTO series (id, title) VALUES ('series-1', 'Test Series');
	`)
	require.NoError(t, err)

	// Run migration
	tx, err := db.Begin()
	require.NoError(t, err)

	migration := &mediaTechInfo{
		migrationBase: NewMigrationBase(21, "media_tech_info"),
	}
	err = migration.Up(tx)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	// Verify movies columns exist with NULL defaults
	t.Run("movies columns exist with NULL defaults", func(t *testing.T) {
		var videoCodec, videoResolution, audioCodec, hdrFormat sql.NullString
		var audioChannels sql.NullInt64
		var subtitleTracks sql.NullString

		err := db.QueryRow(`
			SELECT video_codec, video_resolution, audio_codec, audio_channels, subtitle_tracks, hdr_format
			FROM movies WHERE id = 'movie-1'
		`).Scan(&videoCodec, &videoResolution, &audioCodec, &audioChannels, &subtitleTracks, &hdrFormat)
		require.NoError(t, err)

		assert.False(t, videoCodec.Valid, "video_codec should be NULL")
		assert.False(t, videoResolution.Valid, "video_resolution should be NULL")
		assert.False(t, audioCodec.Valid, "audio_codec should be NULL")
		assert.False(t, audioChannels.Valid, "audio_channels should be NULL")
		assert.False(t, subtitleTracks.Valid, "subtitle_tracks should be NULL")
		assert.False(t, hdrFormat.Valid, "hdr_format should be NULL")
	})

	// Verify series columns exist with NULL defaults (including file_size)
	t.Run("series columns exist with NULL defaults including file_size", func(t *testing.T) {
		var fileSize, audioChannels sql.NullInt64
		var videoCodec, videoResolution, audioCodec, subtitleTracks, hdrFormat sql.NullString

		err := db.QueryRow(`
			SELECT file_size, video_codec, video_resolution, audio_codec, audio_channels, subtitle_tracks, hdr_format
			FROM series WHERE id = 'series-1'
		`).Scan(&fileSize, &videoCodec, &videoResolution, &audioCodec, &audioChannels, &subtitleTracks, &hdrFormat)
		require.NoError(t, err)

		assert.False(t, fileSize.Valid, "file_size should be NULL")
		assert.False(t, videoCodec.Valid, "video_codec should be NULL")
		assert.False(t, videoResolution.Valid, "video_resolution should be NULL")
		assert.False(t, audioCodec.Valid, "audio_codec should be NULL")
		assert.False(t, audioChannels.Valid, "audio_channels should be NULL")
		assert.False(t, subtitleTracks.Valid, "subtitle_tracks should be NULL")
		assert.False(t, hdrFormat.Valid, "hdr_format should be NULL")
	})

	// Verify existing data is preserved
	t.Run("existing data preserved after migration", func(t *testing.T) {
		var title string
		err := db.QueryRow(`SELECT title FROM movies WHERE id = 'movie-1'`).Scan(&title)
		require.NoError(t, err)
		assert.Equal(t, "Test Movie", title)

		err = db.QueryRow(`SELECT title FROM series WHERE id = 'series-1'`).Scan(&title)
		require.NoError(t, err)
		assert.Equal(t, "Test Series", title)
	})

	// Verify columns can be written to
	t.Run("columns are writable", func(t *testing.T) {
		_, err := db.Exec(`
			UPDATE movies SET video_codec = 'H.265', video_resolution = '3840x2160',
				audio_codec = 'DTS', audio_channels = 6,
				subtitle_tracks = '[{"language":"zh-Hant","format":"srt","external":true}]',
				hdr_format = 'HDR10'
			WHERE id = 'movie-1'
		`)
		require.NoError(t, err)

		var videoCodec, videoResolution, audioCodec, hdrFormat, subtitleTracks string
		var audioChannels int
		err = db.QueryRow(`
			SELECT video_codec, video_resolution, audio_codec, audio_channels, subtitle_tracks, hdr_format
			FROM movies WHERE id = 'movie-1'
		`).Scan(&videoCodec, &videoResolution, &audioCodec, &audioChannels, &subtitleTracks, &hdrFormat)
		require.NoError(t, err)

		assert.Equal(t, "H.265", videoCodec)
		assert.Equal(t, "3840x2160", videoResolution)
		assert.Equal(t, "DTS", audioCodec)
		assert.Equal(t, 6, audioChannels)
		assert.Contains(t, subtitleTracks, "zh-Hant")
		assert.Equal(t, "HDR10", hdrFormat)
	})

	// Verify series file_size is writable
	t.Run("series file_size is writable", func(t *testing.T) {
		_, err := db.Exec(`UPDATE series SET file_size = 12345678 WHERE id = 'series-1'`)
		require.NoError(t, err)

		var fileSize int64
		err = db.QueryRow(`SELECT file_size FROM series WHERE id = 'series-1'`).Scan(&fileSize)
		require.NoError(t, err)
		assert.Equal(t, int64(12345678), fileSize)
	})
}

func TestMediaTechInfo_Idempotent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS movies (
			id TEXT PRIMARY KEY, title TEXT NOT NULL,
			genres TEXT NOT NULL DEFAULT '[]', parse_status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS series (
			id TEXT PRIMARY KEY, title TEXT NOT NULL,
			genres TEXT NOT NULL DEFAULT '[]', parse_status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	migration := &mediaTechInfo{
		migrationBase: NewMigrationBase(21, "media_tech_info"),
	}

	// Run migration twice — should not error (columnExists guard)
	tx1, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, migration.Up(tx1))
	require.NoError(t, tx1.Commit())

	tx2, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, migration.Up(tx2))
	require.NoError(t, tx2.Commit())
}

func TestMediaTechInfo_Version(t *testing.T) {
	migration := &mediaTechInfo{
		migrationBase: NewMigrationBase(21, "media_tech_info"),
	}
	assert.Equal(t, int64(21), migration.Version())
	assert.Equal(t, "media_tech_info", migration.Name())
}
