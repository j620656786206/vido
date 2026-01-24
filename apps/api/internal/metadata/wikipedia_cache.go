package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// WikipediaCacheTTL is the default time-to-live for Wikipedia cache entries (7 days per story requirements)
const WikipediaCacheTTL = 7 * 24 * time.Hour

// WikipediaCache provides caching for Wikipedia metadata results
type WikipediaCache struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewWikipediaCache creates a new Wikipedia cache
func NewWikipediaCache(db *sql.DB, logger *slog.Logger) *WikipediaCache {
	if logger == nil {
		logger = slog.Default()
	}

	return &WikipediaCache{
		db:     db,
		logger: logger,
	}
}

// Get retrieves a cached result by query string
func (c *WikipediaCache) Get(ctx context.Context, query string) (*MetadataItem, error) {
	if c.db == nil {
		return nil, nil
	}

	q := `
		SELECT id, page_title, title, original_title, year, director,
		       cast_json, genres_json, summary, image_url, media_type, confidence
		FROM wikipedia_cache
		WHERE query = ? AND expires_at > CURRENT_TIMESTAMP
	`

	var entry wikipediaCacheEntry
	err := c.db.QueryRowContext(ctx, q, query).Scan(
		&entry.ID, &entry.PageTitle, &entry.Title, &entry.OriginalTitle,
		&entry.Year, &entry.Director, &entry.CastJSON, &entry.GenresJSON,
		&entry.Summary, &entry.ImageURL, &entry.MediaType, &entry.Confidence,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Cache miss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query Wikipedia cache: %w", err)
	}

	c.logger.Debug("Wikipedia cache hit",
		"query", query,
		"title", entry.Title,
	)

	return entry.toMetadataItem()
}

// Set stores a result in the cache
func (c *WikipediaCache) Set(ctx context.Context, query string, item *MetadataItem) error {
	if c.db == nil || item == nil {
		return nil
	}

	entry, err := newWikipediaCacheEntry(query, item)
	if err != nil {
		return fmt.Errorf("failed to create Wikipedia cache entry: %w", err)
	}

	q := `
		INSERT OR REPLACE INTO wikipedia_cache (
			id, query, page_title, title, original_title, year, director,
			cast_json, genres_json, summary, image_url, media_type, confidence,
			fetched_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = c.db.ExecContext(ctx, q,
		entry.ID, entry.Query, entry.PageTitle, entry.Title, entry.OriginalTitle,
		entry.Year, entry.Director, entry.CastJSON, entry.GenresJSON,
		entry.Summary, entry.ImageURL, entry.MediaType, entry.Confidence,
		entry.FetchedAt, entry.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert Wikipedia cache entry: %w", err)
	}

	c.logger.Debug("Cached Wikipedia result",
		"query", query,
		"title", item.Title,
		"expires_at", entry.ExpiresAt,
	)

	return nil
}

// GetByPageTitle retrieves a cached result by Wikipedia page title
func (c *WikipediaCache) GetByPageTitle(ctx context.Context, pageTitle string) (*MetadataItem, error) {
	if c.db == nil {
		return nil, nil
	}

	q := `
		SELECT id, page_title, title, original_title, year, director,
		       cast_json, genres_json, summary, image_url, media_type, confidence
		FROM wikipedia_cache
		WHERE page_title = ? AND expires_at > CURRENT_TIMESTAMP
	`

	var entry wikipediaCacheEntry
	err := c.db.QueryRowContext(ctx, q, pageTitle).Scan(
		&entry.ID, &entry.PageTitle, &entry.Title, &entry.OriginalTitle,
		&entry.Year, &entry.Director, &entry.CastJSON, &entry.GenresJSON,
		&entry.Summary, &entry.ImageURL, &entry.MediaType, &entry.Confidence,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query Wikipedia cache by page title: %w", err)
	}

	return entry.toMetadataItem()
}

// Delete removes a cache entry by query
func (c *WikipediaCache) Delete(ctx context.Context, query string) error {
	if c.db == nil {
		return nil
	}

	q := `DELETE FROM wikipedia_cache WHERE query = ?`
	_, err := c.db.ExecContext(ctx, q, query)
	if err != nil {
		return fmt.Errorf("failed to delete Wikipedia cache entry: %w", err)
	}

	return nil
}

// DeleteExpired removes all expired cache entries
func (c *WikipediaCache) DeleteExpired(ctx context.Context) error {
	if c.db == nil {
		return nil
	}

	q := `DELETE FROM wikipedia_cache WHERE expires_at <= CURRENT_TIMESTAMP`
	result, err := c.db.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("failed to delete expired Wikipedia cache entries: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		c.logger.Info("Cleaned up expired Wikipedia cache entries",
			"deleted", affected,
		)
	}

	return nil
}

// Clear removes all cache entries
func (c *WikipediaCache) Clear(ctx context.Context) error {
	if c.db == nil {
		return nil
	}

	q := `DELETE FROM wikipedia_cache`
	_, err := c.db.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("failed to clear Wikipedia cache: %w", err)
	}

	c.logger.Info("Cleared Wikipedia cache")
	return nil
}

// Stats returns cache statistics
func (c *WikipediaCache) Stats(ctx context.Context) (*WikipediaCacheStats, error) {
	if c.db == nil {
		return &WikipediaCacheStats{}, nil
	}

	q := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN expires_at > CURRENT_TIMESTAMP THEN 1 END) as active,
			COUNT(CASE WHEN expires_at <= CURRENT_TIMESTAMP THEN 1 END) as expired
		FROM wikipedia_cache
	`

	var stats WikipediaCacheStats
	err := c.db.QueryRowContext(ctx, q).Scan(&stats.TotalEntries, &stats.ActiveEntries, &stats.ExpiredEntries)
	if err != nil {
		return nil, fmt.Errorf("failed to get Wikipedia cache stats: %w", err)
	}

	stats.TTL = WikipediaCacheTTL
	return &stats, nil
}

// WikipediaCacheStats holds cache statistics
type WikipediaCacheStats struct {
	TotalEntries   int
	ActiveEntries  int
	ExpiredEntries int
	TTL            time.Duration
}

// wikipediaCacheEntry is the internal representation of a cache row
type wikipediaCacheEntry struct {
	ID            string
	Query         string
	PageTitle     string
	Title         string
	OriginalTitle sql.NullString
	Year          sql.NullInt64
	Director      sql.NullString
	CastJSON      sql.NullString
	GenresJSON    sql.NullString
	Summary       sql.NullString
	ImageURL      sql.NullString
	MediaType     string
	Confidence    float64
	FetchedAt     time.Time
	ExpiresAt     time.Time
}

// newWikipediaCacheEntry creates a cache entry from a MetadataItem
func newWikipediaCacheEntry(query string, item *MetadataItem) (*wikipediaCacheEntry, error) {
	entry := &wikipediaCacheEntry{
		ID:         uuid.New().String(),
		Query:      query,
		PageTitle:  item.ID, // Wikipedia page ID is used as the primary ID
		Title:      item.Title,
		MediaType:  string(item.MediaType),
		Confidence: item.Confidence,
		FetchedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(WikipediaCacheTTL),
	}

	if item.OriginalTitle != "" {
		entry.OriginalTitle = sql.NullString{String: item.OriginalTitle, Valid: true}
	}
	if item.Year > 0 {
		entry.Year = sql.NullInt64{Int64: int64(item.Year), Valid: true}
	}
	// Extract director from RawData if available
	if rawData, ok := item.RawData.(map[string]interface{}); ok {
		if infobox, ok := rawData["infobox"]; ok && infobox != nil {
			// Director would be in infobox data
		}
	}
	if len(item.Genres) > 0 {
		genresJSON, _ := json.Marshal(item.Genres)
		entry.GenresJSON = sql.NullString{String: string(genresJSON), Valid: true}
	}
	if item.Overview != "" {
		entry.Summary = sql.NullString{String: item.Overview, Valid: true}
	}
	if item.PosterURL != "" {
		entry.ImageURL = sql.NullString{String: item.PosterURL, Valid: true}
	}

	return entry, nil
}

// toMetadataItem converts a cache entry back to a MetadataItem
func (e *wikipediaCacheEntry) toMetadataItem() (*MetadataItem, error) {
	item := &MetadataItem{
		ID:         e.PageTitle,
		Title:      e.Title,
		TitleZhTW:  e.Title, // Wikipedia zh content is Traditional Chinese
		MediaType:  MediaType(e.MediaType),
		Confidence: e.Confidence,
	}

	if e.OriginalTitle.Valid {
		item.OriginalTitle = e.OriginalTitle.String
	}
	if e.Year.Valid {
		item.Year = int(e.Year.Int64)
	}
	if e.GenresJSON.Valid {
		if err := json.Unmarshal([]byte(e.GenresJSON.String), &item.Genres); err != nil {
			slog.Warn("Failed to unmarshal genres JSON from Wikipedia cache",
				"page_title", e.PageTitle,
				"error", err,
			)
		}
	}
	if e.Summary.Valid {
		item.Overview = e.Summary.String
		item.OverviewZhTW = e.Summary.String
	}
	if e.ImageURL.Valid {
		item.PosterURL = e.ImageURL.String
	}

	return item, nil
}
