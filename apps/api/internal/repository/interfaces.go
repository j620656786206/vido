package repository

import (
	"context"
	"time"

	"github.com/vido/api/internal/models"
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
	ExpiresAt time.Time `db:"expires_at" json:"expiresAt"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
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

// Compile-time interface verification
// These assertions ensure that concrete types implement their respective interfaces.
// If any of these fail to compile, it means the implementation is missing required methods.
var (
	_ MovieRepositoryInterface    = (*MovieRepository)(nil)
	_ SeriesRepositoryInterface   = (*SeriesRepository)(nil)
	_ SettingsRepositoryInterface = (*SettingsRepository)(nil)
	_ CacheRepositoryInterface    = (*CacheRepository)(nil)
	_ SecretsRepositoryInterface  = (*SecretsRepository)(nil)
)
