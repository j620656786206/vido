package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vido/api/internal/config"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// TestDatabasePersistenceAcrossRestarts verifies data persists across database close/reopen cycles
// This integration test ensures the 'survives server restarts' requirement is met
func TestDatabasePersistenceAcrossRestarts(t *testing.T) {
	// Create temporary directory for test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_persistence.db")

	// Test data to write and verify
	testMovie := &models.Movie{
		ID:               "movie-persist-1",
		Title:            "Persistence Test Movie",
		OriginalTitle:    sql.NullString{String: "Original Title", Valid: true},
		ReleaseDate:      "2024-01-01",
		Genres:           []string{"Action", "Drama"},
		Rating:           sql.NullFloat64{Float64: 8.5, Valid: true},
		Overview:         sql.NullString{String: "A movie to test persistence", Valid: true},
		Runtime:          sql.NullInt64{Int64: 120, Valid: true},
		OriginalLanguage: sql.NullString{String: "en", Valid: true},
		Status:           sql.NullString{String: "Released", Valid: true},
		IMDbID:           sql.NullString{String: "tt1234567", Valid: true},
		TMDbID:           sql.NullInt64{Int64: 12345, Valid: true},
	}

	testSeries := &models.Series{
		ID:               "series-persist-1",
		Title:            "Persistence Test Series",
		OriginalTitle:    sql.NullString{String: "Original Series Title", Valid: true},
		FirstAirDate:     sql.NullString{String: "2024-01-01", Valid: true},
		LastAirDate:      sql.NullString{String: "2024-12-31", Valid: true},
		Genres:           []string{"Sci-Fi", "Thriller"},
		Rating:           sql.NullFloat64{Float64: 9.0, Valid: true},
		Overview:         sql.NullString{String: "A series to test persistence", Valid: true},
		NumberOfSeasons:  sql.NullInt64{Int64: 3, Valid: true},
		NumberOfEpisodes: sql.NullInt64{Int64: 30, Valid: true},
		Status:           sql.NullString{String: "Ended", Valid: true},
		OriginalLanguage: sql.NullString{String: "en", Valid: true},
		IMDbID:           sql.NullString{String: "tt7654321", Valid: true},
		TMDbID:           sql.NullInt64{Int64: 54321, Valid: true},
		InProduction:     false,
	}

	testSettingKey := "test_persistence_setting"
	testSettingValue := "persisted_value"

	ctx := context.Background()

	// Phase 1: Create database, write data, and close
	t.Run("Phase1_WriteData", func(t *testing.T) {
		// Create database configuration
		cfg := &config.DatabaseConfig{
			Path:            dbPath,
			WALEnabled:      true,
			WALSyncMode:     "NORMAL",
			WALCheckpoint:   1000,
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 1 * time.Minute,
			BusyTimeout:     5 * time.Second,
			CacheSize:       -64000,
		}

		// Initialize database
		db, err := Initialize(cfg)
		if err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		// Verify WAL mode is enabled
		isWAL, err := db.IsWALEnabled()
		if err != nil {
			t.Fatalf("Failed to check WAL mode: %v", err)
		}
		if !isWAL {
			t.Fatal("Expected WAL mode to be enabled")
		}

		// Run migrations
		runner, err := migrations.NewRunner(db.Conn())
		if err != nil {
			db.Close()
			t.Fatalf("Failed to create migration runner: %v", err)
		}

		// Register all migrations from the global registry
		allMigrations := migrations.GetAll()
		if err := runner.RegisterAll(allMigrations); err != nil {
			db.Close()
			t.Fatalf("Failed to register migrations: %v", err)
		}

		// Run migrations
		if err := runner.Up(ctx); err != nil {
			db.Close()
			t.Fatalf("Failed to run migrations: %v", err)
		}

		// Create repositories
		movieRepo := repository.NewMovieRepository(db.Conn())
		seriesRepo := repository.NewSeriesRepository(db.Conn())
		settingsRepo := repository.NewSettingsRepository(db.Conn())

		// Write test movie
		if err := movieRepo.Create(ctx, testMovie); err != nil {
			db.Close()
			t.Fatalf("Failed to create test movie: %v", err)
		}

		// Write test series
		if err := seriesRepo.Create(ctx, testSeries); err != nil {
			db.Close()
			t.Fatalf("Failed to create test series: %v", err)
		}

		// Write test setting
		if err := settingsRepo.Set(ctx, testSettingKey, testSettingValue, "string"); err != nil {
			db.Close()
			t.Fatalf("Failed to set test setting: %v", err)
		}

		// Verify data was written
		movie, err := movieRepo.FindByID(ctx, testMovie.ID)
		if err != nil {
			db.Close()
			t.Fatalf("Failed to find movie after creation: %v", err)
		}
		if movie.Title != testMovie.Title {
			db.Close()
			t.Errorf("Movie title mismatch: expected %s, got %s", testMovie.Title, movie.Title)
		}

		// Close database connection
		if err := db.Close(); err != nil {
			t.Fatalf("Failed to close database: %v", err)
		}

		// Verify database file exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Fatal("Database file does not exist after close")
		}

		// Verify WAL files exist (indicates WAL mode was active)
		walPath := dbPath + "-wal"
		if _, err := os.Stat(walPath); err == nil {
			t.Log("WAL file exists (normal for WAL mode)")
		}
	})

	// Phase 2: Reopen database and verify data persisted
	t.Run("Phase2_VerifyPersistence", func(t *testing.T) {
		// Create database configuration (same as before)
		cfg := &config.DatabaseConfig{
			Path:            dbPath,
			WALEnabled:      true,
			WALSyncMode:     "NORMAL",
			WALCheckpoint:   1000,
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 1 * time.Minute,
			BusyTimeout:     5 * time.Second,
			CacheSize:       -64000,
		}

		// Initialize database (reopen existing database)
		db, err := Initialize(cfg)
		if err != nil {
			t.Fatalf("Failed to reopen database: %v", err)
		}
		defer db.Close()

		// Verify WAL mode is still enabled
		isWAL, err := db.IsWALEnabled()
		if err != nil {
			t.Fatalf("Failed to check WAL mode after reopen: %v", err)
		}
		if !isWAL {
			t.Fatal("Expected WAL mode to still be enabled after reopen")
		}

		// Create repositories
		movieRepo := repository.NewMovieRepository(db.Conn())
		seriesRepo := repository.NewSeriesRepository(db.Conn())
		settingsRepo := repository.NewSettingsRepository(db.Conn())

		// Verify movie persisted
		movie, err := movieRepo.FindByID(ctx, testMovie.ID)
		if err != nil {
			t.Fatalf("Failed to find movie after reopen: %v", err)
		}
		if movie.Title != testMovie.Title {
			t.Errorf("Movie title mismatch after reopen: expected %s, got %s", testMovie.Title, movie.Title)
		}
		if movie.TMDbID.Int64 != testMovie.TMDbID.Int64 {
			t.Errorf("Movie TMDbID mismatch after reopen: expected %d, got %d", testMovie.TMDbID.Int64, movie.TMDbID.Int64)
		}
		if len(movie.Genres) != len(testMovie.Genres) {
			t.Errorf("Movie genres count mismatch after reopen: expected %d, got %d", len(testMovie.Genres), len(movie.Genres))
		} else {
			for i, genre := range movie.Genres {
				if genre != testMovie.Genres[i] {
					t.Errorf("Movie genre[%d] mismatch after reopen: expected %s, got %s", i, testMovie.Genres[i], genre)
				}
			}
		}

		// Verify series persisted
		series, err := seriesRepo.FindByID(ctx, testSeries.ID)
		if err != nil {
			t.Fatalf("Failed to find series after reopen: %v", err)
		}
		if series.Title != testSeries.Title {
			t.Errorf("Series title mismatch after reopen: expected %s, got %s", testSeries.Title, series.Title)
		}
		if series.NumberOfSeasons.Int64 != testSeries.NumberOfSeasons.Int64 {
			t.Errorf("Series seasons mismatch after reopen: expected %d, got %d", testSeries.NumberOfSeasons.Int64, series.NumberOfSeasons.Int64)
		}
		if series.InProduction != testSeries.InProduction {
			t.Errorf("Series InProduction mismatch after reopen: expected %v, got %v", testSeries.InProduction, series.InProduction)
		}

		// Verify setting persisted
		setting, err := settingsRepo.Get(ctx, testSettingKey)
		if err != nil {
			t.Fatalf("Failed to get setting after reopen: %v", err)
		}
		if setting.Value != testSettingValue {
			t.Errorf("Setting value mismatch after reopen: expected %s, got %s", testSettingValue, setting.Value)
		}

		// Verify migrations were persisted
		var count int
		err = db.Conn().QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query schema_migrations: %v", err)
		}
		if count == 0 {
			t.Error("Expected schema_migrations to have records after reopen")
		}
	})

	// Phase 3: Multiple restart cycles
	t.Run("Phase3_MultipleRestarts", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Path:            dbPath,
			WALEnabled:      true,
			WALSyncMode:     "NORMAL",
			WALCheckpoint:   1000,
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 1 * time.Minute,
			BusyTimeout:     5 * time.Second,
			CacheSize:       -64000,
		}

		// Perform 3 open/close cycles
		for cycle := 1; cycle <= 3; cycle++ {
			db, err := Initialize(cfg)
			if err != nil {
				t.Fatalf("Cycle %d: Failed to open database: %v", cycle, err)
			}

			// Verify data still exists
			movieRepo := repository.NewMovieRepository(db.Conn())
			movie, err := movieRepo.FindByID(ctx, testMovie.ID)
			if err != nil {
				db.Close()
				t.Fatalf("Cycle %d: Failed to find movie: %v", cycle, err)
			}
			if movie.Title != testMovie.Title {
				db.Close()
				t.Errorf("Cycle %d: Movie title changed: expected %s, got %s", cycle, testMovie.Title, movie.Title)
			}

			// Close database
			if err := db.Close(); err != nil {
				t.Fatalf("Cycle %d: Failed to close database: %v", cycle, err)
			}
		}
	})
}

// TestWALModePersistence specifically tests WAL mode persistence
func TestWALModePersistence(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_wal_persistence.db")

	cfg := &config.DatabaseConfig{
		Path:            dbPath,
		WALEnabled:      true,
		WALSyncMode:     "FULL",
		WALCheckpoint:   500,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		BusyTimeout:     5 * time.Second,
		CacheSize:       -64000,
	}

	// Create and configure database
	db, err := Initialize(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Verify WAL mode and sync mode
	isWAL, err := db.IsWALEnabled()
	if err != nil {
		db.Close()
		t.Fatalf("Failed to check WAL mode: %v", err)
	}
	if !isWAL {
		db.Close()
		t.Fatal("Expected WAL mode to be enabled")
	}

	syncMode, err := db.GetSyncMode()
	if err != nil {
		db.Close()
		t.Fatalf("Failed to get sync mode: %v", err)
	}
	expectedSyncMode := "2" // FULL = 2
	if syncMode != expectedSyncMode {
		db.Close()
		t.Errorf("Expected sync mode %s, got %s", expectedSyncMode, syncMode)
	}

	// Close database
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	// Reopen and verify WAL mode persisted
	db, err = Initialize(cfg)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db.Close()

	isWAL, err = db.IsWALEnabled()
	if err != nil {
		t.Fatalf("Failed to check WAL mode after reopen: %v", err)
	}
	if !isWAL {
		t.Fatal("Expected WAL mode to persist after reopen")
	}

	// Verify journal mode is still WAL
	walMode, err := db.GetWALMode()
	if err != nil {
		t.Fatalf("Failed to get WAL mode after reopen: %v", err)
	}
	if walMode != "wal" {
		t.Errorf("Expected journal mode 'wal' after reopen, got '%s'", walMode)
	}
}
