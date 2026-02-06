package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/config"
	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/events"
	"github.com/vido/api/internal/handlers"
	"github.com/vido/api/internal/images"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/secrets"
	"github.com/vido/api/internal/services"

	// Media config is loaded during service initialization
	// and validates directories from VIDO_MEDIA_DIRS env var

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
	mediaService := services.NewMediaService(cfg.MediaDirs)

	// Initialize TMDb service with cache integration (Story 2.1)
	tmdbService := services.NewTMDbService(services.TMDbConfig{
		APIKey:            cfg.TMDbAPIKey,
		DefaultLanguage:   cfg.TMDbDefaultLanguage,
		FallbackLanguages: cfg.TMDbFallbackLanguages,
		CacheTTLHours:     cfg.TMDbCacheTTLHours,
	}, repos.Cache)

	// Initialize AI service for AI-powered filename parsing (Story 3.1)
	aiService, err := services.NewAIService(cfg, db.Conn())
	if err != nil {
		slog.Error("Failed to initialize AI service", "error", err)
		os.Exit(1)
	}
	if aiService != nil {
		slog.Info("AI service initialized", "provider", aiService.GetProviderName())
	} else {
		slog.Info("AI service not configured - AI parsing disabled")
	}

	// Initialize learning service for filename pattern learning (Story 3.9)
	learningService := services.NewLearningService(repos.Learning)
	slog.Info("Learning service initialized")

	// Initialize parser service with AI and learning integration (Story 2.5, 3.1, 3.9)
	// Note: must use a typed nil interface to avoid Go's nil interface gotcha.
	// A nil *AIService assigned to AIServiceInterface creates a non-nil interface
	// (has type but nil value), causing panics on method calls.
	var parserAI services.AIServiceInterface
	if aiService != nil {
		parserAI = aiService
	}
	parserService := services.NewParserServiceWithLearning(parserAI, learningService)

	// Initialize metadata service with multi-source fallback chain (Story 3.3)
	metadataService := services.NewMetadataService(services.MetadataServiceConfig{
		TMDbImageBaseURL:               "https://image.tmdb.org/t/p/w500",
		EnableDouban:                   cfg.EnableDouban,
		EnableWikipedia:                cfg.EnableWikipedia,
		EnableCircuitBreaker:           cfg.EnableCircuitBreaker,
		FallbackDelayMs:                cfg.FallbackDelayMs,
		CircuitBreakerFailureThreshold: cfg.CircuitBreakerFailureThreshold,
		CircuitBreakerTimeoutSeconds:   cfg.CircuitBreakerTimeoutSeconds,
	}, tmdbService)

	// Initialize metadata editor service for manual editing (Story 3.8)
	posterDir := filepath.Join(cfg.DataDir, "posters")
	imageProcessor, err := images.NewImageProcessor(posterDir)
	if err != nil {
		slog.Error("Failed to initialize image processor", "error", err)
		os.Exit(1)
	}
	editService := services.NewMetadataEditService(repos.Movies, repos.Series, imageProcessor)
	metadataService.SetMetadataEditors(editService, editService)
	metadataService.SetPosterUploader(editService)
	slog.Info("Metadata editor initialized", "poster_dir", posterDir)

	// Initialize keyword service and wire to metadata service (Story 3.6)
	if aiService != nil {
		keywordService := services.NewKeywordService(aiService)
		metadataService.SetKeywordGenerator(keywordService)
		slog.Info("AI keyword retry phase enabled for metadata search")
	}

	slog.Info("Services initialized with repository injection")

	// Initialize event emitter for real-time parse progress (Story 3.10)
	parseEventEmitter := events.NewChannelEmitter()
	defer parseEventEmitter.Close()

	// Initialize handlers with injected service interfaces
	// Following Handler → Service → Repository → Database architecture
	movieHandler := handlers.NewMovieHandler(movieService)
	seriesHandler := handlers.NewSeriesHandler(seriesService)
	settingsHandler := handlers.NewSettingsHandler(settingsService)
	mediaHandler := handlers.NewMediaHandler(mediaService)
	tmdbHandler := handlers.NewTMDbHandler(tmdbService)
	parserHandler := handlers.NewParserHandler(parserService)
	metadataHandler := handlers.NewMetadataHandler(metadataService)
	learningHandler := handlers.NewLearningHandler(learningService)
	parseProgressHandler := handlers.NewParseProgressHandler(parseEventEmitter)
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
		mediaHandler.RegisterRoutes(apiV1)
		tmdbHandler.RegisterRoutes(apiV1)
		parserHandler.RegisterRoutes(apiV1)
		metadataHandler.RegisterRoutes(apiV1)
		learningHandler.RegisterRoutes(apiV1)
		parseProgressHandler.RegisterRoutes(apiV1)
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
