package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/config"
	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/handlers"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/secrets"
	"github.com/vido/api/internal/services"

	// Import migrations to register them via init()
	_ "github.com/vido/api/internal/database/migrations"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Validate configuration (fail fast)
	if err := cfg.Validate(); err != nil {
		slog.Error("Configuration validation failed", "error", err)
		os.Exit(1)
	}

	// Log configuration sources
	cfg.LogConfigSources()

	// Set Gin mode based on environment
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	slog.Info("Initializing database", "path", cfg.Database.Path)
	db, err := database.Initialize(cfg.Database)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("Database initialized successfully", "wal_mode", cfg.Database.WALEnabled)

	// Run database migrations
	slog.Info("Running database migrations...")
	migrationRunner, err := migrations.NewRunner(db.Conn())
	if err != nil {
		slog.Error("Failed to create migration runner", "error", err)
		os.Exit(1)
	}

	// Register all migrations from global registry
	allMigrations := migrations.GetAll()
	if err := migrationRunner.RegisterAll(allMigrations); err != nil {
		slog.Error("Failed to register migrations", "error", err)
		os.Exit(1)
	}

	// Apply pending migrations
	ctx := context.Background()
	if err := migrationRunner.Up(ctx); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Check migration status
	status, err := migrationRunner.Status(ctx)
	if err != nil {
		slog.Error("Failed to get migration status", "error", err)
		os.Exit(1)
	}
	appliedCount := 0
	for _, s := range status {
		if s.Applied {
			appliedCount++
		}
	}
	slog.Info("Database migrations completed", "applied", appliedCount, "total", len(status))

	// Initialize repositories via factory (enables future database migration)
	repos := repository.NewRepositoriesWithCache(db.Conn())
	slog.Info("Repositories initialized via factory")

	// Initialize secrets service for encrypted API key storage
	// Uses ENCRYPTION_KEY env var or falls back to machine ID
	secretsService, err := secrets.NewSecretsServiceWithKeyDerivation(repos.Secrets)
	if err != nil {
		slog.Error("Failed to initialize secrets service", "error", err)
		os.Exit(1)
	}
	slog.Info("Secrets service initialized")

	// Initialize services with injected repository interfaces
	// This layered architecture enables testing with mock repositories
	movieService := services.NewMovieService(repos.Movies)
	seriesService := services.NewSeriesService(repos.Series)
	settingsService := services.NewSettingsServiceWithSecrets(repos.Settings, secretsService)
	slog.Info("Services initialized with repository injection")

	// Initialize handlers with injected service interfaces
	// Following Handler → Service → Repository → Database architecture
	movieHandler := handlers.NewMovieHandler(movieService)
	seriesHandler := handlers.NewSeriesHandler(seriesService)
	settingsHandler := handlers.NewSettingsHandler(settingsService)
	slog.Info("Handlers initialized with service injection")

	// Create Gin router
	router := gin.Default()

	// Configure CORS middleware using config values
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.CORSOrigins
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	router.Use(cors.New(corsConfig))
	slog.Info("CORS configured", "origins", cfg.CORSOrigins)

	// Register routes
	router.GET("/health", handlers.HealthCheckHandler(db))

	// API v1 routes with handler → service → repository architecture
	apiV1 := router.Group("/api/v1")
	{
		movieHandler.RegisterRoutes(apiV1)
		seriesHandler.RegisterRoutes(apiV1)
		settingsHandler.RegisterRoutes(apiV1)
	}
	slog.Info("API routes registered", "prefix", "/api/v1")

	// Start server in a goroutine for graceful shutdown
	addr := cfg.GetAddress()
	slog.Info("Starting Vido API server", "address", addr)

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Run server in goroutine
	go func() {
		if err := router.Run(addr); err != nil {
			slog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-quit
	slog.Info("Shutting down server...")

	// Give ongoing requests time to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close database connection
	slog.Info("Closing database connection...")
	if err := db.Close(); err != nil {
		slog.Error("Error closing database", "error", err)
	}

	<-shutdownCtx.Done()
	slog.Info("Server stopped gracefully")
}
