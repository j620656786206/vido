#!/usr/bin/env bash
#
# seed-test-data.sh — Seed deterministic media fixtures for TestSprite journey tests.
#
# WHY: The TestSprite v4 catalog (testsprite_frontend_test_plan.json) needs known
# data on the target server. New P0 cases TC084-086 (Media Detail Panel) require
# a matched/owned movie, a pending-metadata item, and a failed-metadata item.
# Tracks sprint-status `retro-8-TS2` (the execution prerequisite) and the
# test-design doc _bmad-output/test-design-testsprite-coverage-2026-06.md §6.
#
# WHAT THIS SEEDS (directly into the SQLite DB — there is NO create-API for media;
# media normally enters only via the scanner):
#   - 1 media_libraries row (+1 path)          → library context for the items
#   - 1 MATCHED/OWNED movie (tmdb_id set, file_path + tech info + subtitle found)
#                                              → TC084 core metadata, TC085 tech badges + file info
#   - 1 PENDING-metadata movie (parse_status=pending, tmdb_id NULL, has file)
#                                              → TC086 FallbackPending
#   - 1 FAILED-metadata movie (parse_status=failed, tmdb_id NULL, has file)
#                                              → TC086 FallbackFailed
#   All rows use the id prefix "seed-" and a reserved tmdb_id range (9999xxxx) so the
#   script is idempotent (re-runnable) and CANNOT collide with or clobber real data.
#
# WHAT THIS CANNOT SEED (and why — do NOT add fake rows for these):
#   - Downloads Monitor cases (TC079-083): downloads are read LIVE from qBittorrent
#     (download_service.GetAllDownloads → client.GetTorrents); there is no downloads
#     table. To exercise them, add real torrents to the connected qBittorrent — and
#     for TC083 (parse-failed) add a torrent whose name does not parse (e.g. a random
#     string). The parse result is persisted (parse_jobs); the torrent itself is not.
#   - Connection degraded state (TC088): the health indicator is COMPUTED from the
#     live qBittorrent connection. To force "degraded", point the qBittorrent settings
#     host at an unreachable address (or stop qBittorrent) so the health monitor
#     reports degraded, then run TC088.
#
# NOTE: This seeds the panel data the cases read from the DB. The full /media/$type/$id
# route may re-fetch TMDb by id; the reserved 9999xxxx tmdb_ids will not resolve there.
# TC084-086 target the side panel (opened from the grid), which renders stored data.
#
# USAGE:
#   VIDO_DB_PATH=/path/to/vido.db scripts/seed-test-data.sh          # explicit (recommended)
#   scripts/seed-test-data.sh                                        # tries common paths
#   On the NAS, run inside the API container or against the mounted data volume.
#
set -euo pipefail

# --- locate the SQLite database -------------------------------------------------
DB="${VIDO_DB_PATH:-}"
if [[ -z "${DB}" ]]; then
  for c in ./vido.db ./apps/api/vido.db ./data/vido.db /data/vido.db /app/data/vido.db; do
    [[ -f "${c}" ]] && DB="${c}" && break
  done
fi
if [[ -z "${DB}" || ! -f "${DB}" ]]; then
  echo "ERROR: vido.db not found. Set VIDO_DB_PATH=/path/to/vido.db" >&2
  exit 1
fi
command -v sqlite3 >/dev/null 2>&1 || { echo "ERROR: sqlite3 not installed" >&2; exit 1; }

# sanity: expected schema present
if ! sqlite3 "${DB}" "SELECT 1 FROM movies LIMIT 1;" >/dev/null 2>&1; then
  echo "ERROR: '${DB}' has no 'movies' table — run DB migrations first." >&2
  exit 1
fi

echo "Seeding test data into: ${DB}"

# --- idempotent reseed (only ever touches seed- rows; never real data) ----------
sqlite3 "${DB}" <<'SQL'
PRAGMA foreign_keys = OFF;
BEGIN;

DELETE FROM movies              WHERE id LIKE 'seed-%';
DELETE FROM media_library_paths WHERE id LIKE 'seed-%' OR library_id LIKE 'seed-%';
DELETE FROM media_libraries     WHERE id LIKE 'seed-%';

-- Library context -------------------------------------------------------------
INSERT INTO media_libraries (id, name, content_type, auto_detect, sort_order)
VALUES ('seed-lib-movies', 'Seed Movies', 'movie', 0, 0);

INSERT INTO media_library_paths (id, library_id, path, status)
VALUES ('seed-path-movies', 'seed-lib-movies', '/seed/test-movies', 'accessible');

-- (1) MATCHED / OWNED movie — TC084 (title/year/rating) + TC085 (tech badges + file info)
INSERT INTO movies (
  id, title, original_title, release_date, genres, rating, vote_average, vote_count,
  overview, poster_path, backdrop_path, runtime, original_language, tmdb_id,
  file_path, file_size, parse_status, metadata_source, is_removed,
  subtitle_status, subtitle_language, subtitle_search_score, library_id,
  video_codec, video_resolution, audio_codec, audio_channels, hdr_format
) VALUES (
  'seed-movie-owned', 'Seed Owned Movie', 'Seed Owned Movie', '2021-07-30',
  '["Action","Science Fiction"]', 8.1, 8.1, 12345,
  'A fully-matched owned movie seeded for TestSprite TC084/TC085.',
  '/seed_poster.jpg', '/seed_backdrop.jpg', 128, 'en', 99990001,
  '/seed/test-movies/Seed.Owned.Movie.2021.1080p.BluRay.x264.mkv', 8589934592,
  'success', 'tmdb', 0,
  'found', 'zh-Hant', 0.95, 'seed-lib-movies',
  'h264', '1080p', 'eac3', 6, 'none'
);

-- (2) PENDING-metadata movie — TC086 FallbackPending (tmdb_id NULL => unmatched)
INSERT INTO movies (
  id, title, release_date, genres, parse_status, is_removed,
  file_path, file_size, subtitle_status, library_id
) VALUES (
  'seed-movie-pending', 'Seed.Pending.File.2023.1080p.WEB-DL.mkv', '', '[]',
  'pending', 0,
  '/seed/test-movies/Seed.Pending.File.2023.1080p.WEB-DL.mkv', 4294967296,
  'not_searched', 'seed-lib-movies'
);

-- (3) FAILED-metadata movie — TC086 FallbackFailed (tmdb_id NULL => unmatched)
INSERT INTO movies (
  id, title, release_date, genres, parse_status, is_removed,
  file_path, file_size, subtitle_status, library_id
) VALUES (
  'seed-movie-failed', 'Seed.Unparseable.xyz9999.mkv', '', '[]',
  'failed', 0,
  '/seed/test-movies/Seed.Unparseable.xyz9999.mkv', 1073741824,
  'not_searched', 'seed-lib-movies'
);

COMMIT;
SQL

# --- report ---------------------------------------------------------------------
COUNT=$(sqlite3 "${DB}" "SELECT COUNT(*) FROM movies WHERE id LIKE 'seed-%';")
echo "✅ Seeded ${COUNT} movies + 1 library (id prefix 'seed-')."
echo
echo "Covers TestSprite: TC084 (detail core metadata), TC085 (tech badges + file info),"
echo "                   TC086 (fallback pending + failed)."
echo
echo "NOT covered by this script (qBT-live / computed — see header):"
echo "  TC079-083 Downloads Monitor  → add torrents to qBittorrent (parse-failed = unparseable name)"
echo "  TC088     Degraded health     → point qBittorrent host at an unreachable address"
echo
echo "To remove all seed data:  sqlite3 '${DB}' \"DELETE FROM movies WHERE id LIKE 'seed-%';"
echo "                          DELETE FROM media_library_paths WHERE library_id LIKE 'seed-%';"
echo "                          DELETE FROM media_libraries WHERE id LIKE 'seed-%';\""
