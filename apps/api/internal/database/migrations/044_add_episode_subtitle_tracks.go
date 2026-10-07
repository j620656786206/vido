package migrations

import "database/sql"

func init() {
	Register(&addEpisodeSubtitleTracks{
		migrationBase: NewMigrationBase(44, "add_episode_subtitle_tracks"),
	})
}

// addEpisodeSubtitleTracks (disc-2026-10-episode-list-subtitle-badge-a) gives
// episodes the subtitle_tracks column movies have had since migration 021, so
// "does this episode have Chinese subtitles?" can be answered without running
// ffprobe every time a season is opened.
//
//   - subtitle_tracks: JSON array of services.SubtitleTrack — the embedded
//     tracks ffprobe read plus the sidecar files beside the episode (their
//     language decided by content). Same shape as movies.subtitle_tracks, so
//     models.ChineseSubtitleVerdict and the vido_chinese_subtitle SQL function
//     read it unchanged. NULL = never read, which the verdict treats as
//     "unknown" — never as "no subtitles".
//   - subtitle_tracks_file_sig: "<size>:<mtime unix nanos>" of the file when it
//     was probed. The background sweep re-probes an episode whose file no
//     longer matches it (re-encode, replaced release); NULL = never probed.
//
// NULLABLE, not backfilled here: the sweep fills existing rows in the
// background (a migration that ran ffprobe over a NAS library would block
// startup for an hour).
type addEpisodeSubtitleTracks struct {
	migrationBase
}

func (m *addEpisodeSubtitleTracks) Up(tx *sql.Tx) error {
	for _, col := range []string{"subtitle_tracks", "subtitle_tracks_file_sig"} {
		if columnExists(tx, "episodes", col) {
			continue
		}
		if _, err := tx.Exec("ALTER TABLE episodes ADD COLUMN " + col + " TEXT"); err != nil {
			return err
		}
	}
	return nil
}

func (m *addEpisodeSubtitleTracks) Down(tx *sql.Tx) error {
	// Harmless if left in place; SQLite DROP COLUMN support is version-dependent
	// (mirrors 024/032/034/035/041/043).
	return nil
}
