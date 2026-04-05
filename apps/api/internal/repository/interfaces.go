package repository

import (
	"context"
	"time"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/retry"
)

// MovieRepositoryInterface defines the contract for movie data access operations.
// This interface is database-agnostic, allowing future migration from SQLite to PostgreSQL.
type MovieRepositoryInterface interface {
	// Create inserts a new movie into the database
	Create(ctx context.Context, movie *models.Movie) error

	// FindByID retrieves a movie by its primary key
	FindByID(ctx context.Context, id string) (*models.Movie, error)

	// FindByTMDbID retrieves a movie by its TMDb ID
	FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error)

	// FindByIMDbID retrieves a movie by its IMDb ID
	FindByIMDbID(ctx context.Context, imdbID string) (*models.Movie, error)

	// Update modifies an existing movie in the database
	Update(ctx context.Context, movie *models.Movie) error

	// Delete removes a movie from the database by ID
	Delete(ctx context.Context, id string) error

	// List retrieves movies with pagination support
	List(ctx context.Context, params ListParams) ([]models.Movie, *PaginationResult, error)

	// SearchByTitle searches for movies by title with pagination
	SearchByTitle(ctx context.Context, title string, params ListParams) ([]models.Movie, *PaginationResult, error)

	// FullTextSearch performs FTS5 search across title, original_title, and overview
	// Returns movies matching the query with pagination
	FullTextSearch(ctx context.Context, query string, params ListParams) ([]models.Movie, *PaginationResult, error)

	// Upsert creates or updates a movie based on TMDb ID
	// If a movie with the same TMDb ID exists, it updates the existing record
	Upsert(ctx context.Context, movie *models.Movie) error

	// FindByFilePath retrieves a movie by its file path (for duplicate detection)
	FindByFilePath(ctx context.Context, filePath string) (*models.Movie, error)

	// GetDistinctGenres returns all unique genres from movies
	GetDistinctGenres(ctx context.Context) ([]string, error)

	// GetYearRange returns the min and max release years from movies
	GetYearRange(ctx context.Context) (minYear, maxYear int, err error)

	// Count returns the total number of movies
	Count(ctx context.Context) (int, error)

	// BulkCreate inserts multiple movies in a single transaction
	// Needed by: Epic 7 (scan results batch insert)
	BulkCreate(ctx context.Context, movies []*models.Movie) error

	// FindByParseStatus retrieves movies matching a given parse status
	// Needed by: Epic 7 (find pending/failed items for re-scan)
	FindByParseStatus(ctx context.Context, status models.ParseStatus) ([]models.Movie, error)

	// UpdateSubtitleStatus updates subtitle-related fields for a movie
	// Needed by: Epic 8 (subtitle search results)
	UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error

	// FindBySubtitleStatus retrieves movies matching a given subtitle status
	// Needed by: Epic 8 (find items needing subtitles)
	FindBySubtitleStatus(ctx context.Context, status models.SubtitleStatus) ([]models.Movie, error)

	// FindNeedingSubtitleSearch retrieves movies not yet searched or last searched before threshold
	// Needed by: Epic 8 (batch subtitle search scheduler)
	FindNeedingSubtitleSearch(ctx context.Context, olderThan time.Time) ([]models.Movie, error)

	// FindAllWithFilePath retrieves all movies that have a non-null file_path and are not removed
	// Needed by: Story 7-2 (detect removed files during incremental scan)
	FindAllWithFilePath(ctx context.Context) ([]models.Movie, error)

	// GetStats returns aggregate statistics including total and unmatched counts
	// Needed by: Story 9c-4 (unmatched filter count badge)
	GetStats(ctx context.Context) (*MediaStats, error)
}

// SeriesRepositoryInterface defines the contract for TV series data access operations.
// This interface is database-agnostic, allowing future migration from SQLite to PostgreSQL.
type SeriesRepositoryInterface interface {
	// Create inserts a new series into the database
	Create(ctx context.Context, series *models.Series) error

	// FindByID retrieves a series by its primary key
	FindByID(ctx context.Context, id string) (*models.Series, error)

	// FindByTMDbID retrieves a series by its TMDb ID
	FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error)

	// FindByIMDbID retrieves a series by its IMDb ID
	FindByIMDbID(ctx context.Context, imdbID string) (*models.Series, error)

	// Update modifies an existing series in the database
	Update(ctx context.Context, series *models.Series) error

	// Delete removes a series from the database by ID
	Delete(ctx context.Context, id string) error

	// List retrieves series with pagination support
	List(ctx context.Context, params ListParams) ([]models.Series, *PaginationResult, error)

	// SearchByTitle searches for series by title with pagination
	SearchByTitle(ctx context.Context, title string, params ListParams) ([]models.Series, *PaginationResult, error)

	// FullTextSearch performs FTS5 search across title, original_title, and overview
	// Returns series matching the query with pagination
	FullTextSearch(ctx context.Context, query string, params ListParams) ([]models.Series, *PaginationResult, error)

	// Upsert creates or updates a series based on TMDb ID
	// If a series with the same TMDb ID exists, it updates the existing record
	Upsert(ctx context.Context, series *models.Series) error

	// GetDistinctGenres returns all unique genres from series
	GetDistinctGenres(ctx context.Context) ([]string, error)

	// GetYearRange returns the min and max first_air_date years from series
	GetYearRange(ctx context.Context) (minYear, maxYear int, err error)

	// Count returns the total number of series
	Count(ctx context.Context) (int, error)

	// BulkCreate inserts multiple series in a single transaction
	// Needed by: Epic 7 (scan results batch insert)
	BulkCreate(ctx context.Context, seriesList []*models.Series) error

	// FindByParseStatus retrieves series matching a given parse status
	// Needed by: Epic 7 (find pending/failed items for re-scan)
	FindByParseStatus(ctx context.Context, status models.ParseStatus) ([]models.Series, error)

	// UpdateSubtitleStatus updates subtitle-related fields for a series
	// Needed by: Epic 8 (subtitle search results)
	UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error

	// FindBySubtitleStatus retrieves series matching a given subtitle status
	// Needed by: Epic 8 (find items needing subtitles)
	FindBySubtitleStatus(ctx context.Context, status models.SubtitleStatus) ([]models.Series, error)

	// FindNeedingSubtitleSearch retrieves series not yet searched or last searched before threshold
	// Needed by: Epic 8 (batch subtitle search scheduler)
	FindNeedingSubtitleSearch(ctx context.Context, olderThan time.Time) ([]models.Series, error)

	// GetStats returns aggregate statistics including total and unmatched counts
	// Needed by: Story 9c-4 (unmatched filter count badge)
	GetStats(ctx context.Context) (*MediaStats, error)
}

// SeasonRepositoryInterface defines the contract for season data access operations.
type SeasonRepositoryInterface interface {
	// Create inserts a new season into the database
	Create(ctx context.Context, season *models.Season) error

	// FindByID retrieves a season by its primary key
	FindByID(ctx context.Context, id string) (*models.Season, error)

	// FindBySeriesID retrieves all seasons for a series
	FindBySeriesID(ctx context.Context, seriesID string) ([]models.Season, error)

	// FindBySeriesAndNumber retrieves a season by series ID and season number
	FindBySeriesAndNumber(ctx context.Context, seriesID string, seasonNumber int) (*models.Season, error)

	// Update modifies an existing season in the database
	Update(ctx context.Context, season *models.Season) error

	// Delete removes a season from the database by ID
	Delete(ctx context.Context, id string) error

	// Upsert creates or updates a season based on UNIQUE(series_id, season_number)
	Upsert(ctx context.Context, season *models.Season) error
}

// EpisodeRepositoryInterface defines the contract for episode data access operations.
type EpisodeRepositoryInterface interface {
	// Create inserts a new episode into the database
	Create(ctx context.Context, episode *models.Episode) error

	// FindByID retrieves an episode by its primary key
	FindByID(ctx context.Context, id string) (*models.Episode, error)

	// FindBySeriesID retrieves all episodes for a series
	FindBySeriesID(ctx context.Context, seriesID string) ([]models.Episode, error)

	// FindBySeasonID retrieves all episodes for a specific season by season ID
	FindBySeasonID(ctx context.Context, seasonID string) ([]models.Episode, error)

	// FindBySeasonNumber retrieves all episodes for a specific season of a series
	FindBySeasonNumber(ctx context.Context, seriesID string, seasonNumber int) ([]models.Episode, error)

	// FindBySeriesSeasonEpisode retrieves an episode by series ID, season, and episode number
	FindBySeriesSeasonEpisode(ctx context.Context, seriesID string, season, episode int) (*models.Episode, error)

	// Update modifies an existing episode in the database
	Update(ctx context.Context, episode *models.Episode) error

	// Delete removes an episode from the database by ID
	Delete(ctx context.Context, id string) error

	// Upsert creates or updates an episode based on series_id, season_number, episode_number
	Upsert(ctx context.Context, episode *models.Episode) error
}

// SettingsRepositoryInterface defines the contract for application settings data access.
// This interface is database-agnostic, allowing future migration from SQLite to PostgreSQL.
type SettingsRepositoryInterface interface {
	// Set creates or updates a setting (upsert operation)
	Set(ctx context.Context, setting *models.Setting) error

	// Get retrieves a setting by its key
	Get(ctx context.Context, key string) (*models.Setting, error)

	// GetAll retrieves all settings
	GetAll(ctx context.Context) ([]models.Setting, error)

	// Delete removes a setting from the database by key
	Delete(ctx context.Context, key string) error

	// GetString retrieves a setting as a string value
	GetString(ctx context.Context, key string) (string, error)

	// GetInt retrieves a setting as an integer value
	GetInt(ctx context.Context, key string) (int, error)

	// GetBool retrieves a setting as a boolean value
	GetBool(ctx context.Context, key string) (bool, error)

	// SetString is a convenience method to set a string value
	SetString(ctx context.Context, key, value string) error

	// SetInt is a convenience method to set an integer value
	SetInt(ctx context.Context, key string, value int) error

	// SetBool is a convenience method to set a boolean value
	SetBool(ctx context.Context, key string, value bool) error
}

// CacheEntry represents a cached item with TTL support
type CacheEntry struct {
	Key       string    `db:"key" json:"key"`
	Value     string    `db:"value" json:"value"`
	Type      string    `db:"type" json:"type"` // "tmdb", "ai", "image", etc.
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// CacheRepositoryInterface defines the contract for cache data access operations.
// This interface supports TTL-based expiration for different cache types.
type CacheRepositoryInterface interface {
	// Get retrieves a cache entry by key, returns nil if not found or expired
	Get(ctx context.Context, key string) (*CacheEntry, error)

	// Set creates or updates a cache entry with the specified TTL
	Set(ctx context.Context, key string, value string, cacheType string, ttl time.Duration) error

	// Delete removes a cache entry by key
	Delete(ctx context.Context, key string) error

	// Clear removes all cache entries
	Clear(ctx context.Context) error

	// ClearExpired removes all expired cache entries
	ClearExpired(ctx context.Context) (int64, error)

	// ClearByType removes all cache entries of a specific type
	ClearByType(ctx context.Context, cacheType string) (int64, error)
}

// SecretsRepositoryInterface defines the contract for encrypted secrets data access.
// This interface provides storage for secrets encrypted with AES-256-GCM.
type SecretsRepositoryInterface interface {
	// Set creates or updates an encrypted secret (upsert by name)
	Set(ctx context.Context, name string, encryptedValue string) error

	// Get retrieves an encrypted secret by name
	Get(ctx context.Context, name string) (string, error)

	// Delete removes a secret by name
	Delete(ctx context.Context, name string) error

	// Exists checks if a secret with the given name exists
	Exists(ctx context.Context, name string) (bool, error)

	// List returns all secret names (not values)
	List(ctx context.Context) ([]string, error)
}

// LearningRepositoryInterface defines the contract for learning pattern storage
type LearningRepositoryInterface interface {
	Save(ctx context.Context, mapping *models.FilenameMapping) error
	FindByID(ctx context.Context, id string) (*models.FilenameMapping, error)
	FindByExactPattern(ctx context.Context, pattern string) (*models.FilenameMapping, error)
	FindByFansubAndTitle(ctx context.Context, fansubGroup, titlePattern string) ([]*models.FilenameMapping, error)
	ListWithRegex(ctx context.Context) ([]*models.FilenameMapping, error)
	ListAll(ctx context.Context) ([]*models.FilenameMapping, error)
	Update(ctx context.Context, mapping *models.FilenameMapping) error
	Delete(ctx context.Context, id string) error
	IncrementUseCount(ctx context.Context, id string) error
	Count(ctx context.Context) (int, error)
}

// RetryRepositoryInterface defines the contract for retry queue data access
type RetryRepositoryInterface interface {
	Add(ctx context.Context, item *RetryItem) error
	FindByID(ctx context.Context, id string) (*RetryItem, error)
	FindByTaskID(ctx context.Context, taskID string) (*RetryItem, error)
	GetPending(ctx context.Context, now time.Time) ([]*RetryItem, error)
	GetAll(ctx context.Context) ([]*RetryItem, error)
	Update(ctx context.Context, item *RetryItem) error
	Delete(ctx context.Context, id string) error
	DeleteByTaskID(ctx context.Context, taskID string) error
	Count(ctx context.Context) (int, error)
	CountByTaskType(ctx context.Context, taskType string) (int, error)
	ClearAll(ctx context.Context) error
	// Stats methods for tracking historical retry data (Story 3.11)
	IncrementQueued(ctx context.Context, taskType string) error
	IncrementSucceeded(ctx context.Context, taskType string) error
	IncrementFailed(ctx context.Context, taskType string) error
	IncrementExhausted(ctx context.Context, taskType string) error
	GetStats(ctx context.Context) (*retry.RetryStats, error)
}

// ParseJobRepositoryInterface defines the contract for parse job data access operations.
type ParseJobRepositoryInterface interface {
	// Create inserts a new parse job into the database
	Create(ctx context.Context, job *models.ParseJob) error

	// GetByID retrieves a parse job by its primary key
	GetByID(ctx context.Context, id string) (*models.ParseJob, error)

	// GetByTorrentHash retrieves a parse job by torrent hash
	GetByTorrentHash(ctx context.Context, hash string) (*models.ParseJob, error)

	// GetPending retrieves pending parse jobs with a limit
	GetPending(ctx context.Context, limit int) ([]*models.ParseJob, error)

	// UpdateStatus updates a parse job's status and optional error message
	UpdateStatus(ctx context.Context, id string, status models.ParseJobStatus, errMsg string) error

	// Update modifies an existing parse job in the database
	Update(ctx context.Context, job *models.ParseJob) error

	// Delete removes a parse job from the database by ID
	Delete(ctx context.Context, id string) error

	// ListAll retrieves all parse jobs ordered by creation time descending
	ListAll(ctx context.Context, limit int) ([]*models.ParseJob, error)
}

// RetryItem is imported from retry package for interface definition
type RetryItem = retry.RetryItem

// LogRepositoryInterface defines the contract for system log data access operations.
type LogRepositoryInterface interface {
	// GetLogs retrieves paginated system logs with optional filters
	GetLogs(ctx context.Context, filter models.LogFilter) ([]models.SystemLog, int, error)

	// CreateLog inserts a new log entry
	CreateLog(ctx context.Context, log *models.SystemLog) error

	// CreateLogBatch inserts multiple log entries in a single transaction
	CreateLogBatch(ctx context.Context, logs []models.SystemLog) error

	// DeleteOlderThan removes logs older than the specified number of days
	DeleteOlderThan(ctx context.Context, days int) (int64, error)
}

// BackupRepositoryInterface defines the contract for backup data access operations.
type BackupRepositoryInterface interface {
	// Create inserts a new backup record
	Create(ctx context.Context, backup *models.Backup) error

	// List retrieves all backups ordered by creation time descending
	List(ctx context.Context) ([]models.Backup, error)

	// GetByID retrieves a backup by its ID
	GetByID(ctx context.Context, id string) (*models.Backup, error)

	// Update modifies an existing backup record
	Update(ctx context.Context, backup *models.Backup) error

	// Delete removes a backup record by ID
	Delete(ctx context.Context, id string) error

	// TotalSizeBytes returns the sum of all completed backup sizes
	TotalSizeBytes(ctx context.Context) (int64, error)
}

// Compile-time interface verification
// These assertions ensure that concrete types implement their respective interfaces.
// If any of these fail to compile, it means the implementation is missing required methods.
var (
	_ MovieRepositoryInterface    = (*MovieRepository)(nil)
	_ SeriesRepositoryInterface   = (*SeriesRepository)(nil)
	_ SeasonRepositoryInterface   = (*SeasonRepository)(nil)
	_ EpisodeRepositoryInterface  = (*EpisodeRepository)(nil)
	_ SettingsRepositoryInterface = (*SettingsRepository)(nil)
	_ CacheRepositoryInterface    = (*CacheRepository)(nil)
	_ SecretsRepositoryInterface  = (*SecretsRepository)(nil)
	_ LearningRepositoryInterface = (*LearningRepository)(nil)
	_ RetryRepositoryInterface    = (*RetryRepository)(nil)
	_ LogRepositoryInterface      = (*LogRepository)(nil)
	_ BackupRepositoryInterface   = (*BackupRepository)(nil)
)
