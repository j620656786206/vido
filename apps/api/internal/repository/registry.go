package repository

import (
	"database/sql"
)

// Repositories holds all repository interfaces for dependency injection.
// This struct enables swapping implementations (e.g., SQLite to PostgreSQL)
// without changing the service layer code.
type Repositories struct {
	Movies   MovieRepositoryInterface
	Series   SeriesRepositoryInterface
	Settings SettingsRepositoryInterface
	Cache    CacheRepositoryInterface
	Secrets  SecretsRepositoryInterface
	Learning *LearningRepository
}

// NewRepositories creates all repository implementations for the given database connection.
// This factory function is the single point for initializing data access layer.
//
// Usage:
//
//	db, _ := sql.Open("sqlite", "vido.db")
//	repos := repository.NewRepositories(db)
//	movieService := services.NewMovieService(repos.Movies)
func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		Movies:   NewMovieRepository(db),
		Series:   NewSeriesRepository(db),
		Settings: NewSettingsRepository(db),
		// Cache will be initialized after CacheRepository implementation in Task 4
		Cache:    nil,
		Secrets:  NewSecretsRepository(db),
		Learning: NewLearningRepository(db),
	}
}

// SetCacheRepository sets the cache repository after initialization.
// This allows the cache repository to be added after the migration is applied.
func (r *Repositories) SetCacheRepository(cache CacheRepositoryInterface) {
	r.Cache = cache
}

// NewRepositoriesWithCache creates all repository implementations including cache.
// Use this after the cache_entries table migration has been applied.
func NewRepositoriesWithCache(db *sql.DB) *Repositories {
	return &Repositories{
		Movies:   NewMovieRepository(db),
		Series:   NewSeriesRepository(db),
		Settings: NewSettingsRepository(db),
		Cache:    NewCacheRepository(db),
		Secrets:  NewSecretsRepository(db),
		Learning: NewLearningRepository(db),
	}
}
