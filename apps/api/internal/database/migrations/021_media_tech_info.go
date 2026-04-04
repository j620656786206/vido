package migrations

import "database/sql"

func init() {
	Register(&mediaTechInfo{
		migrationBase: NewMigrationBase(21, "media_tech_info"),
	})
}

type mediaTechInfo struct {
	migrationBase
}

func (m *mediaTechInfo) Up(tx *sql.Tx) error {
	// Add technical info columns to movies table
	movieColumns := []struct {
		column string
		ddl    string
	}{
		{"video_codec", "ALTER TABLE movies ADD COLUMN video_codec TEXT"},
		{"video_resolution", "ALTER TABLE movies ADD COLUMN video_resolution TEXT"},
		{"audio_codec", "ALTER TABLE movies ADD COLUMN audio_codec TEXT"},
		{"audio_channels", "ALTER TABLE movies ADD COLUMN audio_channels INTEGER"},
		{"subtitle_tracks", "ALTER TABLE movies ADD COLUMN subtitle_tracks TEXT"},
		{"hdr_format", "ALTER TABLE movies ADD COLUMN hdr_format TEXT"},
	}

	for _, mc := range movieColumns {
		if !columnExists(tx, "movies", mc.column) {
			if _, err := tx.Exec(mc.ddl); err != nil {
				return err
			}
		}
	}

	// Add technical info columns + file_size to series table
	seriesColumns := []struct {
		column string
		ddl    string
	}{
		{"file_size", "ALTER TABLE series ADD COLUMN file_size INTEGER"},
		{"video_codec", "ALTER TABLE series ADD COLUMN video_codec TEXT"},
		{"video_resolution", "ALTER TABLE series ADD COLUMN video_resolution TEXT"},
		{"audio_codec", "ALTER TABLE series ADD COLUMN audio_codec TEXT"},
		{"audio_channels", "ALTER TABLE series ADD COLUMN audio_channels INTEGER"},
		{"subtitle_tracks", "ALTER TABLE series ADD COLUMN subtitle_tracks TEXT"},
		{"hdr_format", "ALTER TABLE series ADD COLUMN hdr_format TEXT"},
	}

	for _, sc := range seriesColumns {
		if !columnExists(tx, "series", sc.column) {
			if _, err := tx.Exec(sc.ddl); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *mediaTechInfo) Down(tx *sql.Tx) error {
	// SQLite does not support DROP COLUMN before 3.35.0
	// For safety, this is a no-op — columns are nullable and harmless if unused
	return nil
}
