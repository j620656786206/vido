package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/zhtw"
)

// ChineseMetadataBackfill turns metadata already stored in Simplified Chinese
// Traditional (disc-2026-10-metadata-zh-fallback-simplified AC #4). The TMDb
// layer now converts what it borrows from zh-CN, but rows written before that
// keep 钢铁之泪 until something rewrites them — this is that something.
//
// It runs once: a settings marker is written when every table converted
// cleanly, and later starts return at once (the TMDb layer converts anything
// new). A failed run leaves no marker and is retried at the next start; it is
// idempotent, since a converted field is no longer Simplified.
//
// Which fields are Simplified is zhtw.SimplifiedFields — the subtitle
// detector's ~200-character table misses short titles (大象之梦, 霸王别姬 read
// "undetermined"), and s2tw alone changed Traditional text on a copy of the
// production library (蘇里→蘇裡, 好傢伙→好傢夥).
//
// Only TMDb metadata is touched: rows whose metadata_source is tmdb, and the
// seasons/episodes of such series. A manual edit or the user's .nfo is theirs;
// an unmatched row's title is its file name; Douban converts on its own.
type ChineseMetadataBackfill struct {
	db       *sql.DB
	conv     zhtw.ScriptConverter
	cache    TMDbCacheClearer
	settings ChineseBackfillMarker
	logger   *slog.Logger
}

// TMDbCacheClearer drops cached TMDb answers, which may still hold Simplified
// text for up to the cache TTL — a rescan inside that window would store it
// again.
type TMDbCacheClearer interface {
	ClearByType(ctx context.Context, cacheType string) (int64, error)
}

// ChineseBackfillMarker remembers that the backfill has run.
type ChineseBackfillMarker interface {
	GetBool(ctx context.Context, key string) (bool, error)
	SetBool(ctx context.Context, key string, value bool) error
}

const (
	// chineseBackfillDoneKey marks a backfill that converted every table.
	chineseBackfillDoneKey = "metadata.zh_hant_backfill_done"
	// chineseBackfillBatch caps the fields in one OpenCC run.
	chineseBackfillBatch = 200
	// tmdbCacheType is tmdb.CacheTypeTMDb.
	tmdbCacheType = "tmdb"
)

// NewChineseMetadataBackfill wires the backfill. conv nil disables it.
func NewChineseMetadataBackfill(db *sql.DB, conv zhtw.ScriptConverter, cache TMDbCacheClearer, settings ChineseBackfillMarker, logger *slog.Logger) *ChineseMetadataBackfill {
	if logger == nil {
		logger = slog.Default()
	}
	return &ChineseMetadataBackfill{db: db, conv: conv, cache: cache, settings: settings, logger: logger}
}

// backfillTable is one table's read and write. The SELECT returns id, the two
// text fields (named by cols) and the production_countries JSON that picks the
// wording rule; seasons and episodes take it from their series.
type backfillTable struct {
	name  string
	query string
	cols  [2]string
}

// fromTMDb keeps the rows whose metadata came from TMDb (alias = the row that
// carries metadata_source).
func fromTMDb(alias string) string {
	return fmt.Sprintf("%s.metadata_source = '%s'", alias, models.MetadataSourceTMDb)
}

func chineseBackfillTables() []backfillTable {
	return []backfillTable{
		{
			name: "movies",
			query: `SELECT m.id, m.title, COALESCE(m.overview, ''), COALESCE(m.production_countries, '')
				FROM movies m WHERE ` + fromTMDb("m"),
			cols: [2]string{"title", "overview"},
		},
		{
			name: "series",
			query: `SELECT s.id, s.title, COALESCE(s.overview, ''), COALESCE(s.production_countries, '')
				FROM series s WHERE ` + fromTMDb("s"),
			cols: [2]string{"title", "overview"},
		},
		{
			name: "seasons",
			query: `SELECT x.id, COALESCE(x.name, ''), COALESCE(x.overview, ''), COALESCE(s.production_countries, '')
				FROM seasons x JOIN series s ON s.id = x.series_id WHERE ` + fromTMDb("s"),
			cols: [2]string{"name", "overview"},
		},
		{
			name: "episodes",
			query: `SELECT x.id, COALESCE(x.title, ''), COALESCE(x.overview, ''), COALESCE(s.production_countries, '')
				FROM episodes x JOIN series s ON s.id = x.series_id WHERE ` + fromTMDb("s"),
			cols: [2]string{"title", "overview"},
		},
	}
}

// backfillRow is one row's two fields, and what they become.
type backfillRow struct {
	id        any
	fields    [2]string
	converted [2]string
	countries []string
}

// Run converts every table and logs how many rows each one changed. Errors in
// one table are logged and the next table still runs; the returned error is
// the first one, and only a clean run is marked done.
func (b *ChineseMetadataBackfill) Run(ctx context.Context) error {
	if b.conv == nil {
		b.logger.Warn("Simplified metadata backfill skipped: no OpenCC converter")
		return nil
	}
	if b.settings != nil {
		if done, err := b.settings.GetBool(ctx, chineseBackfillDoneKey); err == nil && done {
			return nil
		}
	}
	var firstErr error
	counts := map[string]int{}
	for _, t := range chineseBackfillTables() {
		n, err := b.runTable(ctx, t)
		counts[t.name] = n
		if err != nil {
			b.logger.Warn("Simplified metadata backfill failed for a table", "table", t.name, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	b.logger.Info("Simplified metadata backfill finished",
		"movies", counts["movies"], "series", counts["series"],
		"seasons", counts["seasons"], "episodes", counts["episodes"])
	if firstErr != nil {
		return firstErr
	}
	if b.clearTMDbCache(ctx) && b.settings != nil {
		if err := b.settings.SetBool(ctx, chineseBackfillDoneKey, true); err != nil {
			b.logger.Warn("Simplified metadata backfill: could not record that it ran", "error", err)
		}
	}
	return nil
}

// clearTMDbCache drops the cached TMDb answers: those cached before the fix
// still carry Simplified text, and a rescan inside the cache TTL would store
// it again. It reports whether the backfill may be marked done.
func (b *ChineseMetadataBackfill) clearTMDbCache(ctx context.Context) bool {
	if b.cache == nil {
		return true
	}
	n, err := b.cache.ClearByType(ctx, tmdbCacheType)
	if err != nil {
		b.logger.Warn("Simplified metadata backfill: could not clear the TMDb cache", "error", err)
		return false
	}
	b.logger.Info("Simplified metadata backfill: cleared cached TMDb answers", "entries", n)
	return true
}

func (b *ChineseMetadataBackfill) runTable(ctx context.Context, t backfillTable) (int, error) {
	rows, err := b.readRows(ctx, t)
	if err != nil {
		return 0, err
	}
	changed, err := b.convertRows(rows)
	if err != nil {
		return 0, err
	}
	if len(changed) == 0 {
		return 0, nil
	}
	// One transaction per table: thousands of episodes are one commit, and a
	// failure leaves the table as it was. Only the fields that changed are
	// written, so an empty overview stays exactly as stored.
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin %s: %w", t.name, err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, r := range changed {
		for i, col := range t.cols {
			if r.converted[i] == r.fields[i] {
				continue
			}
			//nolint:gosec // table and column names are the constants above
			q := fmt.Sprintf("UPDATE %s SET %s = ? WHERE id = ?", t.name, col)
			if _, err := tx.ExecContext(ctx, q, r.converted[i], r.id); err != nil {
				return 0, fmt.Errorf("update %s %v: %w", t.name, r.id, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit %s: %w", t.name, err)
	}
	return len(changed), nil
}

func (b *ChineseMetadataBackfill) readRows(ctx context.Context, t backfillTable) ([]*backfillRow, error) {
	rs, err := b.db.QueryContext(ctx, t.query)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", t.name, err)
	}
	defer rs.Close()
	var out []*backfillRow
	for rs.Next() {
		r := &backfillRow{}
		var countriesJSON string
		if err := rs.Scan(&r.id, &r.fields[0], &r.fields[1], &countriesJSON); err != nil {
			return nil, fmt.Errorf("scan %s: %w", t.name, err)
		}
		if !hasHan(r.fields[0]) && !hasHan(r.fields[1]) {
			continue
		}
		r.countries = countryCodes(countriesJSON)
		out = append(out, r)
	}
	return out, rs.Err()
}

// convertRows returns the rows that had Simplified text, with converted set.
// A field that is not Simplified keeps its stored value.
func (b *ChineseMetadataBackfill) convertRows(rows []*backfillRow) ([]*backfillRow, error) {
	type field struct {
		row *backfillRow
		i   int
	}
	var fields []field
	var texts []string
	for _, r := range rows {
		r.converted = r.fields
		for i, f := range r.fields {
			if f != "" {
				fields = append(fields, field{r, i})
				texts = append(texts, f)
			}
		}
	}

	var own, taiwan []field
	for start := 0; start < len(texts); start += chineseBackfillBatch {
		end := min(start+chineseBackfillBatch, len(texts))
		simplified, err := zhtw.SimplifiedFields(b.conv, texts[start:end])
		if err != nil {
			return nil, fmt.Errorf("opencc: %w", err)
		}
		for k, yes := range simplified {
			if !yes {
				continue
			}
			f := fields[start+k]
			if zhtw.KeepsOwnWording(f.row.countries) {
				own = append(own, f)
			} else {
				taiwan = append(taiwan, f)
			}
		}
	}

	changed := map[*backfillRow]bool{}
	for _, g := range []struct {
		fields    []field
		countries []string
	}{{own, []string{"CN"}}, {taiwan, nil}} {
		for start := 0; start < len(g.fields); start += chineseBackfillBatch {
			chunk := g.fields[start:min(start+chineseBackfillBatch, len(g.fields))]
			src := make([]string, len(chunk))
			for k, f := range chunk {
				src[k] = f.row.fields[f.i]
			}
			out, err := zhtw.ConvertJoined(func(text string) (string, error) {
				return zhtw.Finalize(b.conv, text, g.countries)
			}, src)
			if err != nil {
				return nil, fmt.Errorf("opencc: %w", err)
			}
			for k, f := range chunk {
				f.row.converted[f.i] = out[k]
				changed[f.row] = true
			}
		}
	}

	var out []*backfillRow
	for _, r := range rows {
		if changed[r] {
			out = append(out, r)
		}
	}
	return out, nil
}

// countryCodes reads the ISO codes out of a stored production_countries JSON.
func countryCodes(raw string) []string {
	if raw == "" {
		return nil
	}
	var pcs []models.ProductionCountry
	if err := json.Unmarshal([]byte(raw), &pcs); err != nil {
		return nil
	}
	codes := make([]string, 0, len(pcs))
	for _, pc := range pcs {
		codes = append(codes, pc.ISO3166_1)
	}
	return codes
}
