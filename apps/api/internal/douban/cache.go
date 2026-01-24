package douban

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CacheConfig holds configuration for the Douban cache
type CacheConfig struct {
	// DefaultTTL is the default time-to-live for cache entries (default: 7 days)
	DefaultTTL time.Duration
	// CleanupInterval is how often to clean up expired entries (default: 1 hour)
	CleanupInterval time.Duration
	// Enabled controls whether caching is active
	Enabled bool
}

// DefaultCacheConfig returns the default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		DefaultTTL:      7 * 24 * time.Hour, // 7 days per story requirements
		CleanupInterval: 1 * time.Hour,
		Enabled:         true,
	}
}

// Cache provides caching for Douban scraping results
type Cache struct {
	db     *sql.DB
	config CacheConfig
	logger *slog.Logger

	stopCleanup chan struct{}
	wg          sync.WaitGroup
}

// NewCache creates a new Douban cache
func NewCache(db *sql.DB, config CacheConfig, logger *slog.Logger) *Cache {
	if logger == nil {
		logger = slog.Default()
	}

	c := &Cache{
		db:          db,
		config:      config,
		logger:      logger,
		stopCleanup: make(chan struct{}),
	}

	// Start background cleanup goroutine
	if config.Enabled && config.CleanupInterval > 0 {
		c.wg.Add(1)
		go c.cleanupLoop()
	}

	return c
}

// Close stops the background cleanup goroutine
func (c *Cache) Close() {
	close(c.stopCleanup)
	c.wg.Wait()
}

// cleanupLoop periodically removes expired cache entries
func (c *Cache) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.DeleteExpired(context.Background()); err != nil {
				c.logger.Warn("Failed to cleanup expired Douban cache entries",
					"error", err,
				)
			}
		case <-c.stopCleanup:
			return
		}
	}
}

// Get retrieves a cached result by Douban ID
func (c *Cache) Get(ctx context.Context, doubanID string) (*DetailResult, error) {
	if !c.config.Enabled || c.db == nil {
		return nil, nil
	}

	query := `
		SELECT id, douban_id, title, title_traditional, original_title, year,
		       rating, rating_count, director, cast_json, genres_json,
		       countries_json, languages_json, poster_url, summary,
		       summary_traditional, media_type, runtime, episodes,
		       release_date, imdb_id, scraped_at
		FROM douban_cache
		WHERE douban_id = ? AND expires_at > CURRENT_TIMESTAMP
	`

	var entry cacheEntry
	err := c.db.QueryRowContext(ctx, query, doubanID).Scan(
		&entry.ID, &entry.DoubanID, &entry.Title, &entry.TitleTraditional,
		&entry.OriginalTitle, &entry.Year, &entry.Rating, &entry.RatingCount,
		&entry.Director, &entry.CastJSON, &entry.GenresJSON, &entry.CountriesJSON,
		&entry.LanguagesJSON, &entry.PosterURL, &entry.Summary,
		&entry.SummaryTraditional, &entry.MediaType, &entry.Runtime,
		&entry.Episodes, &entry.ReleaseDate, &entry.IMDbID, &entry.ScrapedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query cache: %w", err)
	}

	c.logger.Debug("Cache hit",
		"douban_id", doubanID,
		"title", entry.Title,
	)

	return entry.toDetailResult()
}

// Set stores a result in the cache
func (c *Cache) Set(ctx context.Context, result *DetailResult) error {
	if !c.config.Enabled || c.db == nil || result == nil {
		return nil
	}

	entry, err := newCacheEntry(result, c.config.DefaultTTL)
	if err != nil {
		return fmt.Errorf("failed to create cache entry: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO douban_cache (
			id, douban_id, title, title_traditional, original_title, year,
			rating, rating_count, director, cast_json, genres_json,
			countries_json, languages_json, poster_url, summary,
			summary_traditional, media_type, runtime, episodes,
			release_date, imdb_id, scraped_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = c.db.ExecContext(ctx, query,
		entry.ID, entry.DoubanID, entry.Title, entry.TitleTraditional,
		entry.OriginalTitle, entry.Year, entry.Rating, entry.RatingCount,
		entry.Director, entry.CastJSON, entry.GenresJSON, entry.CountriesJSON,
		entry.LanguagesJSON, entry.PosterURL, entry.Summary,
		entry.SummaryTraditional, entry.MediaType, entry.Runtime,
		entry.Episodes, entry.ReleaseDate, entry.IMDbID, entry.ScrapedAt,
		entry.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert cache entry: %w", err)
	}

	c.logger.Debug("Cached Douban result",
		"douban_id", result.ID,
		"title", result.Title,
		"expires_at", entry.ExpiresAt,
	)

	return nil
}

// GetByTitle searches the cache by title
func (c *Cache) GetByTitle(ctx context.Context, title string) ([]*DetailResult, error) {
	if !c.config.Enabled || c.db == nil {
		return nil, nil
	}

	query := `
		SELECT id, douban_id, title, title_traditional, original_title, year,
		       rating, rating_count, director, cast_json, genres_json,
		       countries_json, languages_json, poster_url, summary,
		       summary_traditional, media_type, runtime, episodes,
		       release_date, imdb_id, scraped_at
		FROM douban_cache
		WHERE (title LIKE ? OR title_traditional LIKE ?) AND expires_at > CURRENT_TIMESTAMP
		ORDER BY rating DESC
		LIMIT 10
	`

	pattern := "%" + title + "%"
	rows, err := c.db.QueryContext(ctx, query, pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query cache by title: %w", err)
	}
	defer rows.Close()

	var results []*DetailResult
	for rows.Next() {
		var entry cacheEntry
		err := rows.Scan(
			&entry.ID, &entry.DoubanID, &entry.Title, &entry.TitleTraditional,
			&entry.OriginalTitle, &entry.Year, &entry.Rating, &entry.RatingCount,
			&entry.Director, &entry.CastJSON, &entry.GenresJSON, &entry.CountriesJSON,
			&entry.LanguagesJSON, &entry.PosterURL, &entry.Summary,
			&entry.SummaryTraditional, &entry.MediaType, &entry.Runtime,
			&entry.Episodes, &entry.ReleaseDate, &entry.IMDbID, &entry.ScrapedAt,
		)
		if err != nil {
			c.logger.Warn("Failed to scan cache row", "error", err)
			continue
		}

		result, err := entry.toDetailResult()
		if err != nil {
			c.logger.Warn("Failed to convert cache entry", "error", err)
			continue
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cache rows: %w", err)
	}

	return results, nil
}

// Delete removes a cache entry by Douban ID
func (c *Cache) Delete(ctx context.Context, doubanID string) error {
	if c.db == nil {
		return nil
	}

	query := `DELETE FROM douban_cache WHERE douban_id = ?`
	_, err := c.db.ExecContext(ctx, query, doubanID)
	if err != nil {
		return fmt.Errorf("failed to delete cache entry: %w", err)
	}

	return nil
}

// DeleteExpired removes all expired cache entries
func (c *Cache) DeleteExpired(ctx context.Context) error {
	if c.db == nil {
		return nil
	}

	query := `DELETE FROM douban_cache WHERE expires_at <= CURRENT_TIMESTAMP`
	result, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired cache entries: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		c.logger.Info("Cleaned up expired Douban cache entries",
			"deleted", affected,
		)
	}

	return nil
}

// Clear removes all cache entries
func (c *Cache) Clear(ctx context.Context) error {
	if c.db == nil {
		return nil
	}

	query := `DELETE FROM douban_cache`
	_, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	c.logger.Info("Cleared Douban cache")
	return nil
}

// Stats returns cache statistics
func (c *Cache) Stats(ctx context.Context) (*CacheStats, error) {
	if c.db == nil {
		return &CacheStats{}, nil
	}

	query := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN expires_at > CURRENT_TIMESTAMP THEN 1 END) as active,
			COUNT(CASE WHEN expires_at <= CURRENT_TIMESTAMP THEN 1 END) as expired
		FROM douban_cache
	`

	var stats CacheStats
	err := c.db.QueryRowContext(ctx, query).Scan(&stats.TotalEntries, &stats.ActiveEntries, &stats.ExpiredEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache stats: %w", err)
	}

	stats.TTL = c.config.DefaultTTL
	return &stats, nil
}

// CacheStats holds cache statistics
type CacheStats struct {
	TotalEntries   int
	ActiveEntries  int
	ExpiredEntries int
	TTL            time.Duration
}

// cacheEntry is the internal representation of a cache row
type cacheEntry struct {
	ID                 string
	DoubanID           string
	Title              string
	TitleTraditional   sql.NullString
	OriginalTitle      sql.NullString
	Year               sql.NullInt64
	Rating             sql.NullFloat64
	RatingCount        sql.NullInt64
	Director           sql.NullString
	CastJSON           sql.NullString
	GenresJSON         sql.NullString
	CountriesJSON      sql.NullString
	LanguagesJSON      sql.NullString
	PosterURL          sql.NullString
	Summary            sql.NullString
	SummaryTraditional sql.NullString
	MediaType          string
	Runtime            sql.NullInt64
	Episodes           sql.NullInt64
	ReleaseDate        sql.NullString
	IMDbID             sql.NullString
	ScrapedAt          time.Time
	ExpiresAt          time.Time
}

// newCacheEntry creates a cache entry from a DetailResult
func newCacheEntry(result *DetailResult, ttl time.Duration) (*cacheEntry, error) {
	entry := &cacheEntry{
		ID:        uuid.New().String(),
		DoubanID:  result.ID,
		Title:     result.Title,
		MediaType: string(result.Type),
		ScrapedAt: result.ScrapedAt,
		ExpiresAt: time.Now().Add(ttl),
	}

	if result.TitleTraditional != "" {
		entry.TitleTraditional = sql.NullString{String: result.TitleTraditional, Valid: true}
	}
	if result.OriginalTitle != "" {
		entry.OriginalTitle = sql.NullString{String: result.OriginalTitle, Valid: true}
	}
	if result.Year > 0 {
		entry.Year = sql.NullInt64{Int64: int64(result.Year), Valid: true}
	}
	if result.Rating > 0 {
		entry.Rating = sql.NullFloat64{Float64: result.Rating, Valid: true}
	}
	if result.RatingCount > 0 {
		entry.RatingCount = sql.NullInt64{Int64: int64(result.RatingCount), Valid: true}
	}
	if result.Director != "" {
		entry.Director = sql.NullString{String: result.Director, Valid: true}
	}
	if len(result.Cast) > 0 {
		castJSON, _ := json.Marshal(result.Cast)
		entry.CastJSON = sql.NullString{String: string(castJSON), Valid: true}
	}
	if len(result.Genres) > 0 {
		genresJSON, _ := json.Marshal(result.Genres)
		entry.GenresJSON = sql.NullString{String: string(genresJSON), Valid: true}
	}
	if len(result.Countries) > 0 {
		countriesJSON, _ := json.Marshal(result.Countries)
		entry.CountriesJSON = sql.NullString{String: string(countriesJSON), Valid: true}
	}
	if len(result.Languages) > 0 {
		languagesJSON, _ := json.Marshal(result.Languages)
		entry.LanguagesJSON = sql.NullString{String: string(languagesJSON), Valid: true}
	}
	if result.PosterURL != "" {
		entry.PosterURL = sql.NullString{String: result.PosterURL, Valid: true}
	}
	if result.Summary != "" {
		entry.Summary = sql.NullString{String: result.Summary, Valid: true}
	}
	if result.SummaryTraditional != "" {
		entry.SummaryTraditional = sql.NullString{String: result.SummaryTraditional, Valid: true}
	}
	if result.Runtime > 0 {
		entry.Runtime = sql.NullInt64{Int64: int64(result.Runtime), Valid: true}
	}
	if result.Episodes > 0 {
		entry.Episodes = sql.NullInt64{Int64: int64(result.Episodes), Valid: true}
	}
	if result.ReleaseDate != "" {
		entry.ReleaseDate = sql.NullString{String: result.ReleaseDate, Valid: true}
	}
	if result.IMDbID != "" {
		entry.IMDbID = sql.NullString{String: result.IMDbID, Valid: true}
	}

	return entry, nil
}

// toDetailResult converts a cache entry back to a DetailResult
func (e *cacheEntry) toDetailResult() (*DetailResult, error) {
	result := &DetailResult{
		ID:        e.DoubanID,
		Title:     e.Title,
		Type:      MediaType(e.MediaType),
		ScrapedAt: e.ScrapedAt,
	}

	if e.TitleTraditional.Valid {
		result.TitleTraditional = e.TitleTraditional.String
	}
	if e.OriginalTitle.Valid {
		result.OriginalTitle = e.OriginalTitle.String
	}
	if e.Year.Valid {
		result.Year = int(e.Year.Int64)
	}
	if e.Rating.Valid {
		result.Rating = e.Rating.Float64
	}
	if e.RatingCount.Valid {
		result.RatingCount = int(e.RatingCount.Int64)
	}
	if e.Director.Valid {
		result.Director = e.Director.String
	}
	if e.CastJSON.Valid {
		if err := json.Unmarshal([]byte(e.CastJSON.String), &result.Cast); err != nil {
			slog.Warn("Failed to unmarshal cast JSON from cache",
				"douban_id", e.DoubanID,
				"error", err,
			)
		}
	}
	if e.GenresJSON.Valid {
		if err := json.Unmarshal([]byte(e.GenresJSON.String), &result.Genres); err != nil {
			slog.Warn("Failed to unmarshal genres JSON from cache",
				"douban_id", e.DoubanID,
				"error", err,
			)
		}
	}
	if e.CountriesJSON.Valid {
		if err := json.Unmarshal([]byte(e.CountriesJSON.String), &result.Countries); err != nil {
			slog.Warn("Failed to unmarshal countries JSON from cache",
				"douban_id", e.DoubanID,
				"error", err,
			)
		}
	}
	if e.LanguagesJSON.Valid {
		if err := json.Unmarshal([]byte(e.LanguagesJSON.String), &result.Languages); err != nil {
			slog.Warn("Failed to unmarshal languages JSON from cache",
				"douban_id", e.DoubanID,
				"error", err,
			)
		}
	}
	if e.PosterURL.Valid {
		result.PosterURL = e.PosterURL.String
	}
	if e.Summary.Valid {
		result.Summary = e.Summary.String
	}
	if e.SummaryTraditional.Valid {
		result.SummaryTraditional = e.SummaryTraditional.String
	}
	if e.Runtime.Valid {
		result.Runtime = int(e.Runtime.Int64)
	}
	if e.Episodes.Valid {
		result.Episodes = int(e.Episodes.Int64)
	}
	if e.ReleaseDate.Valid {
		result.ReleaseDate = e.ReleaseDate.String
	}
	if e.IMDbID.Valid {
		result.IMDbID = e.IMDbID.String
	}

	return result, nil
}
