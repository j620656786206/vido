package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/repository"
)

// ─── disc-2026-10-metadata-zh-fallback-simplified AC #4 ─────────────────────

// backfillOpenCC stands in for the OpenCC CLI: s2tw swaps characters, s2twp
// also rewrites the phrase 視頻→影片 (as OpenCC's TWPhrases does) so a test can
// tell which wording rule ran.
type backfillOpenCC struct{ err error }

var backfillPairs = []string{"钢", "鋼", "铁", "鐵", "泪", "淚", "梦", "夢", "别", "別",
	"这", "這", "个", "個", "视", "視", "频", "頻", "录", "錄", "华", "華", "里", "裡", "杰", "傑", "苏", "蘇"}

var backfillS2TW = strings.NewReplacer(backfillPairs...)

// backfillT2S is the reverse table.
var backfillT2S = func() *strings.Replacer {
	rev := make([]string, len(backfillPairs))
	for i := 0; i < len(backfillPairs); i += 2 {
		rev[i], rev[i+1] = backfillPairs[i+1], backfillPairs[i]
	}
	return strings.NewReplacer(rev...)
}()

func (c backfillOpenCC) ConvertS2TW(b []byte) ([]byte, error) {
	if c.err != nil {
		return b, c.err
	}
	return []byte(backfillS2TW.Replace(string(b))), nil
}

func (c backfillOpenCC) ConvertT2S(b []byte) ([]byte, error) {
	if c.err != nil {
		return b, c.err
	}
	return []byte(backfillT2S.Replace(string(b))), nil
}

func (c backfillOpenCC) ConvertS2TWP(b []byte) ([]byte, error) {
	if c.err != nil {
		return b, c.err
	}
	return []byte(strings.ReplaceAll(backfillS2TW.Replace(string(b)), "視頻", "影片")), nil
}

func seedBackfillLibrary(t *testing.T, db *sql.DB) {
	t.Helper()
	exec := func(q string, args ...any) {
		t.Helper()
		_, err := db.Exec(q, args...)
		require.NoError(t, err, q)
	}
	now := time.Now()
	movie := `INSERT INTO movies (id, title, overview, metadata_source, production_countries, created_at, updated_at, release_date) VALUES (?, ?, ?, ?, ?, ?, ?, '2012-09-26')`
	// Simplified, borrowed from zh-CN before the fix: a Dutch film.
	exec(movie, "m-tos", "钢铁之泪", "这个视频", "tmdb", `[{"iso_3166_1":"NL","name":"Netherlands"}]`, now, now)
	// Already Traditional — and its overview holds a phrase s2twp WOULD rewrite.
	exec(movie, "m-trad", "鋼鐵之淚", "台灣的視頻平台", "tmdb", `[{"iso_3166_1":"NL","name":"Netherlands"}]`, now, now)
	// Traditional with a character s2tw would still rewrite (傑克蘇里 → 蘇裡
	// on the production copy): t2s changes it, so it is not Simplified.
	exec(movie, "m-name", "藍色星球", "傑克蘇里與奈蒂莉", "tmdb", `[{"iso_3166_1":"US","name":"United States"}]`, now, now)
	// English only.
	exec(movie, "m-en", "Sintel", "A girl and her dragon.", nil, nil, now, now)
	// Unmatched: the title is the file name, never rewritten.
	exec(movie, "m-file", "[钢铁之泪].2012.mkv", "", nil, nil, now, now)
	// The user's own edit, in Simplified on purpose: never touched.
	exec(movie, "m-manual", "霸王别姬", "这个", "manual", nil, now, now)
	// The user's own .nfo: never touched either.
	exec(movie, "m-nfo", "大象之梦", "", "nfo", nil, now, now)
	// A mainland film keeps its own wording: characters only.
	exec(movie, "m-cn", "霸王别姬", "这个视频", "tmdb", `[{"iso_3166_1":"CN","name":"China"}]`, now, now)

	exec(`INSERT INTO series (id, title, overview, metadata_source, production_countries, created_at, updated_at, first_air_date) VALUES (?, ?, ?, ?, ?, ?, ?, '2022-06-02')`,
		"s-1", "梦华录", "这个视频", "tmdb", `[{"iso_3166_1":"CN","name":"China"}]`, now, now)
	exec(`INSERT INTO series (id, title, overview, metadata_source, production_countries, created_at, updated_at, first_air_date) VALUES (?, ?, ?, ?, ?, ?, ?, '2022-06-02')`,
		"s-manual", "梦", "这个", "manual", nil, now, now)
	exec(`INSERT INTO seasons (id, series_id, season_number, name, overview, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"se-1", "s-1", 1, "第一季", "这个视频", now, now)
	exec(`INSERT INTO episodes (id, series_id, season_number, episode_number, title, overview, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"e-1", "s-1", 1, 1, "梦", "视频", now, now)
	exec(`INSERT INTO episodes (id, series_id, season_number, episode_number, title, overview, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"e-manual", "s-manual", 1, 1, "梦", "视频", now, now)
}

func readTitleOverview(t *testing.T, db *sql.DB, table, titleCol, id string) (string, string) {
	t.Helper()
	var title, overview sql.NullString
	require.NoError(t, db.QueryRow(`SELECT `+titleCol+`, overview FROM `+table+` WHERE id = ?`, id).Scan(&title, &overview))
	return title.String, overview.String
}

func TestChineseMetadataBackfill_ConvertsOnlyStoredSimplified(t *testing.T) {
	db := activityLibraryDB(t)
	seedBackfillLibrary(t, db)
	cache := repository.NewCacheRepository(db)
	settings := repository.NewSettingsRepository(db)
	ctx := context.Background()
	require.NoError(t, cache.Set(ctx, "tmdb:movie/133701", `{"title":"钢铁之泪"}`, tmdbCacheType, time.Hour))
	require.NoError(t, cache.Set(ctx, "asr_chunk:keep", `{}`, "asr_chunk", time.Hour))

	b := NewChineseMetadataBackfill(db, backfillOpenCC{}, cache, settings, nil)
	require.NoError(t, b.Run(ctx))

	title, overview := readTitleOverview(t, db, "movies", "title", "m-tos")
	assert.Equal(t, "鋼鐵之淚", title)
	assert.Equal(t, "這個影片", overview, "a Dutch film gets Taiwan phrases")

	title, overview = readTitleOverview(t, db, "movies", "title", "m-trad")
	assert.Equal(t, "鋼鐵之淚", title)
	assert.Equal(t, "台灣的視頻平台", overview, "s2tw leaves it alone, so s2twp never gets to rewrite it")

	_, overview = readTitleOverview(t, db, "movies", "title", "m-name")
	assert.Equal(t, "傑克蘇里與奈蒂莉", overview, "Traditional text s2tw would damage is left alone")

	title, _ = readTitleOverview(t, db, "movies", "title", "m-en")
	assert.Equal(t, "Sintel", title)

	title, _ = readTitleOverview(t, db, "movies", "title", "m-file")
	assert.Equal(t, "[钢铁之泪].2012.mkv", title, "an unmatched row's title is its file name")

	title, overview = readTitleOverview(t, db, "movies", "title", "m-manual")
	assert.Equal(t, "霸王别姬", title, "the user's own edit is never touched")
	assert.Equal(t, "这个", overview)
	title, _ = readTitleOverview(t, db, "movies", "title", "m-nfo")
	assert.Equal(t, "大象之梦", title, "the user's own .nfo is never touched")

	title, overview = readTitleOverview(t, db, "movies", "title", "m-cn")
	assert.Equal(t, "霸王別姬", title)
	assert.Equal(t, "這個視頻", overview, "a mainland film keeps its own wording")

	title, overview = readTitleOverview(t, db, "series", "title", "s-1")
	assert.Equal(t, "夢華錄", title)
	assert.Equal(t, "這個視頻", overview)
	title, overview = readTitleOverview(t, db, "seasons", "name", "se-1")
	assert.Equal(t, "第一季", title)
	assert.Equal(t, "這個視頻", overview, "a season follows its series' wording rule")
	title, overview = readTitleOverview(t, db, "episodes", "title", "e-1")
	assert.Equal(t, "夢", title)
	assert.Equal(t, "視頻", overview)
	title, _ = readTitleOverview(t, db, "episodes", "title", "e-manual")
	assert.Equal(t, "梦", title, "a hand-edited series keeps its episodes as they are")

	entry, err := cache.Get(ctx, "tmdb:movie/133701")
	require.NoError(t, err)
	assert.Nil(t, entry, "cached TMDb answers from before the fix are dropped")
	entry, err = cache.Get(ctx, "asr_chunk:keep")
	require.NoError(t, err)
	assert.NotNil(t, entry, "other cache types are left alone")
}

func TestChineseMetadataBackfill_SecondRunChangesNothing(t *testing.T) {
	db := activityLibraryDB(t)
	seedBackfillLibrary(t, db)
	cache := repository.NewCacheRepository(db)
	settings := repository.NewSettingsRepository(db)
	ctx := context.Background()
	b := NewChineseMetadataBackfill(db, backfillOpenCC{}, cache, settings, nil)
	require.NoError(t, b.Run(ctx))

	snapshot := func() string {
		var sb strings.Builder
		for _, q := range []string{
			`SELECT id || title || COALESCE(overview, '') FROM movies ORDER BY id`,
			`SELECT id || title || COALESCE(overview, '') FROM series ORDER BY id`,
			`SELECT id || COALESCE(name, '') || COALESCE(overview, '') FROM seasons ORDER BY id`,
			`SELECT id || COALESCE(title, '') || COALESCE(overview, '') FROM episodes ORDER BY id`,
		} {
			rows, err := db.Query(q)
			require.NoError(t, err)
			for rows.Next() {
				var s string
				require.NoError(t, rows.Scan(&s))
				sb.WriteString(s + "\n")
			}
			require.NoError(t, rows.Close())
		}
		return sb.String()
	}
	first := snapshot()

	// A TMDb answer cached after the first run must survive the second: the
	// marker makes the second run return at once.
	require.NoError(t, cache.Set(ctx, "tmdb:movie/1", `{}`, tmdbCacheType, time.Hour))
	require.NoError(t, b.Run(ctx))

	assert.Equal(t, first, snapshot())
	entry, err := cache.Get(ctx, "tmdb:movie/1")
	require.NoError(t, err)
	assert.NotNil(t, entry, "the TMDb cache is cleared once, not at every start")
}

func TestChineseMetadataBackfill_ConverterFailureWritesNothing(t *testing.T) {
	db := activityLibraryDB(t)
	seedBackfillLibrary(t, db)

	settings := repository.NewSettingsRepository(db)
	b := NewChineseMetadataBackfill(db, backfillOpenCC{err: errors.New("opencc: exit status 1")}, nil, settings, nil)
	require.Error(t, b.Run(context.Background()))

	title, overview := readTitleOverview(t, db, "movies", "title", "m-tos")
	assert.Equal(t, "钢铁之泪", title, "a failed run leaves the stored text as it was")
	assert.Equal(t, "这个视频", overview)

	// No marker was written, so the next start tries again.
	require.NoError(t, NewChineseMetadataBackfill(db, backfillOpenCC{}, nil, settings, nil).Run(context.Background()))
	title, _ = readTitleOverview(t, db, "movies", "title", "m-tos")
	assert.Equal(t, "鋼鐵之淚", title)
}

func TestChineseMetadataBackfill_NoConverterIsANoop(t *testing.T) {
	db := activityLibraryDB(t)
	seedBackfillLibrary(t, db)

	require.NoError(t, NewChineseMetadataBackfill(db, nil, nil, nil, nil).Run(context.Background()))
	title, _ := readTitleOverview(t, db, "movies", "title", "m-tos")
	assert.Equal(t, "钢铁之泪", title)
}
