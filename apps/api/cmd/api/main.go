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
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/config"
	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/events"
	"github.com/vido/api/internal/handlers"
	"github.com/vido/api/internal/health"
	"github.com/vido/api/internal/images"
	"github.com/vido/api/internal/logger"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/retry"
	"github.com/vido/api/internal/secrets"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/sse"
	"github.com/vido/api/internal/subtitle"
	subtitleproviders "github.com/vido/api/internal/subtitle/providers"
	"github.com/vido/api/internal/cache"

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

	// Initialize DB log handler for system logs (Story 6.3)
	// Must come after migrations so the system_logs table exists
	logRepo := repository.NewLogRepository(db.Conn())
	dbLogHandler := logger.NewDBHandler(logRepo)
	defer dbLogHandler.Close()
	// Create a concrete stdout handler to avoid infinite recursion.
	// slog.Default().Handler() returns a defaultHandler that delegates back to
	// slog.Default(), which would cause a loop after slog.SetDefault(multiHandler).
	stdoutHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	multiHandler := slog.New(logger.NewMultiHandler(stdoutHandler, dbLogHandler))
	slog.SetDefault(multiHandler)
	slog.Info("System log DB handler initialized")

	// Initialize offline cache for graceful degradation (Story 3.12)
	offlineCache := cache.NewOfflineCache(db.Conn())
	if err := offlineCache.InitSchema(ctx); err != nil {
		slog.Error("Failed to initialize offline cache schema", "error", err)
		os.Exit(1)
	}
	slog.Info("Offline cache initialized")

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
	setupService := services.NewSetupService(repos.Settings, secretsService)
	qbittorrentService := services.NewQBittorrentService(repos.Settings, secretsService)
	downloadService := services.NewDownloadService(qbittorrentService, slog.Default())
	mediaService := services.NewMediaService(cfg.MediaDirs)
	mediaLibraryService := services.NewMediaLibraryService(repos.MediaLibraries)
	setupService.SetLibraryService(mediaLibraryService) // Story 7b-3: wire library creation into setup

	// Initialize log service (Story 6.3)
	logService := services.NewLogService(repos.Logs)

	// Initialize backup service (Story 6.5)
	backupDir := filepath.Join(cfg.DataDir, "backups")
	backupService := services.NewBackupService(db.Conn(), repos.Backups, backupDir, 17)
	slog.Info("Backup service initialized", "backup_dir", backupDir)

	// Initialize backup scheduler (Story 6.8)
	backupScheduler := services.NewBackupScheduler(backupService, repos.Settings, repos.Backups)
	slog.Info("Backup scheduler initialized")

	// Initialize export service (Story 6.9)
	exportDir := filepath.Join(cfg.DataDir, "exports")
	exportService := services.NewExportService(repos.Movies, repos.Series, exportDir)
	slog.Info("Export service initialized", "export_dir", exportDir)

	// Initialize cache management services (Story 6.2)
	posterDir := filepath.Join(cfg.DataDir, "posters")
	cacheStatsService := services.NewCacheStatsService(db.Conn(), posterDir)
	cacheCleanupService := services.NewCacheCleanupService(db.Conn(), posterDir)
	slog.Info("Cache management services initialized")

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

	// Initialize health monitoring for graceful degradation (Story 3.12)
	// Use actual service health checks where available, config-based checks for optional services
	var tmdbPingable health.Pingable = tmdbService
	var aiPingable health.Pingable
	if aiService != nil {
		aiPingable = aiService
	}
	doubanPingable := health.NewConfigurablePingable("Douban", cfg.EnableDouban)
	wikipediaPingable := health.NewConfigurablePingable("Wikipedia", cfg.EnableWikipedia)
	healthChecker := health.NewServiceHealthChecker(tmdbPingable, doubanPingable, wikipediaPingable, aiPingable)
	// Wire qBittorrent health check via Ping method on client (Story 4.6)
	qbHealthPingable := health.NewQBPingable(
		qbittorrentService.IsConfigured,
		func(ctx context.Context) error {
			_, err := qbittorrentService.TestConnection(ctx)
			return err
		},
	)
	healthChecker.SetQBittorrent(qbHealthPingable)
	healthMonitor := health.NewHealthMonitor(healthChecker)
	healthMonitor.SetHistoryRepo(repos.ConnectionHistory)
	degradationService := services.NewDegradationServiceWithCache(healthMonitor, offlineCache)
	slog.Info("Health monitoring initialized with service health checks and offline cache")

	// Initialize retry service for auto-retry mechanism (Story 3.11)
	// Note: executor will be wired up after metadata service is created
	// We create a placeholder executor first and update it after metadata service exists
	var retryExecutor *retry.RetryExecutor
	retryService := services.NewRetryService(repos.Retry, nil, slog.Default())
	slog.Info("Retry service initialized (executor pending)")

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

	// Wire up retry service with metadata service (Story 3.11)
	// Create executor that can re-execute failed metadata searches
	retryExecutor = retry.NewRetryExecutor(metadataService, slog.Default())
	// Recreate retry service with the executor now that we have it
	retryService = services.NewRetryService(repos.Retry, retryExecutor, slog.Default())
	// Wire retry service to metadata service for automatic retry queueing
	metadataService.SetRetryService(retryService)

	// Set up retry event handler for notifications and stats tracking (Story 3.11 - AC2, AC3)
	retryService.SetEventHandler(func(event retry.Event) {
		statsCtx := context.Background()
		switch event.Type {
		case retry.EventRetrySuccess:
			slog.Info("Retry succeeded - metadata now available",
				"task_id", event.Item.TaskID,
				"task_type", event.Item.TaskType,
				"attempts", event.Item.AttemptCount,
			)
			// Record success stat
			if err := retryService.RecordSucceeded(statsCtx, event.Item.TaskType); err != nil {
				slog.Warn("Failed to record success stat", "error", err)
			}
			// TODO: Emit SSE event for real-time UI notification (Story 3.11 - AC3)
		case retry.EventRetryExhausted:
			slog.Warn("Retry exhausted - manual intervention required",
				"task_id", event.Item.TaskID,
				"task_type", event.Item.TaskType,
				"attempts", event.Item.AttemptCount,
				"last_error", event.Item.LastError,
			)
			// Record exhausted stat
			if err := retryService.RecordExhausted(statsCtx, event.Item.TaskType); err != nil {
				slog.Warn("Failed to record exhausted stat", "error", err)
			}
			// TODO: Emit SSE event for real-time UI notification (Story 3.11 - AC2)
		case retry.EventRetryFailed:
			slog.Debug("Retry attempt failed, will retry later",
				"task_id", event.Item.TaskID,
				"attempt", event.Item.AttemptCount,
				"next_attempt", event.Metadata["next_attempt"],
			)
			// Record failed attempt stat
			if err := retryService.RecordFailed(statsCtx, event.Item.TaskType); err != nil {
				slog.Warn("Failed to record failed stat", "error", err)
			}
		}
	})
	slog.Info("Retry executor and event handler configured")

	slog.Info("Services initialized with repository injection")

	// Initialize SSE hub for real-time event broadcasting
	sseHub := sse.NewHub()
	defer sseHub.Close()
	slog.Info("SSE hub initialized")

	// Initialize scanner service for media library scanning (Story 7.1)
	scannerService := services.NewScannerService(
		repos.Movies,
		repos.Series,
		cfg.MediaDirs,
		sseHub,
		slog.Default(),
	)
	scannerService.SetLibraryRepo(repos.MediaLibraries) // Story 7b-5: DB-based library scanning
	scannerService.SetEpisodeRepo(repos.Episodes) // Story 9c-3: series file_size aggregation
	slog.Info("Scanner service initialized")

	// Initialize NFO reader service for .nfo sidecar parsing (Story 9c-2)
	nfoReaderService := services.NewNFOReaderService(slog.Default())
	slog.Info("NFO reader service initialized")

	// Initialize FFprobe service for video technical info extraction (Story 9c-3)
	ffprobeService := services.NewFFprobeService(3, 10*time.Second, slog.Default())
	slog.Info("FFprobe service initialized", "available", ffprobeService.IsAvailable())

	// Initialize enrichment service for post-scan metadata enrichment
	enrichmentService := services.NewEnrichmentService(
		repos.Movies,
		parserService,
		metadataService,
		nfoReaderService,
		tmdbService,
		ffprobeService,
		sseHub,
		slog.Default(),
	)
	// Wire post-scan auto-enrichment: after scan completes with new/updated files,
	// automatically trigger metadata enrichment in background
	scannerService.SetOnScanComplete(func() {
		go func() {
			result, err := enrichmentService.StartEnrichment(context.Background())
			if err != nil {
				slog.Error("post-scan enrichment failed", "error", err)
				return
			}
			slog.Info("post-scan enrichment completed",
				"succeeded", result.Succeeded,
				"failed", result.Failed,
				"duration", result.Duration,
			)
		}()
	})
	slog.Info("Enrichment service initialized with post-scan auto-trigger")

	// Initialize scan scheduler (Story 7.2)
	scanScheduler := services.NewScanScheduler(scannerService, repos.Settings, slog.Default())
	slog.Info("Scan scheduler initialized")

	// Initialize subtitle engine components (Story 8.1-8.8)
	subtitleConverter, _ := subtitle.NewConverter()
	subtitleScorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	subtitlePlacer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	// Initialize subtitle providers (Assrt, OpenSubtitles, Zimuku)
	assrtProvider := subtitleproviders.NewAssrtProvider(ctx, secretsService)
	opensubProvider := subtitleproviders.NewOpenSubProvider(ctx, secretsService)
	zimukuProvider := subtitleproviders.NewZimukuProvider()
	subtitleProviders := []subtitleproviders.SubtitleProvider{assrtProvider, opensubProvider, zimukuProvider}
	subtitleEngine := subtitle.NewEngine(
		subtitleProviders, subtitleScorer, subtitleConverter, subtitlePlacer,
		sseHub, repos.Movies, repos.Series,
	)
	// Initialize audio extractor service (Story 9.2a)
	audioExtractorService := services.NewAudioExtractorService(1, 5*time.Minute, slog.Default())
	slog.Info("Audio extractor service initialized", "available", audioExtractorService.IsAvailable())

	// Initialize Whisper client and transcription service (Story 9.2a)
	var transcriptionService *services.TranscriptionService
	if cfg.HasOpenAIKey() && audioExtractorService.IsAvailable() {
		whisperClient := ai.NewWhisperClient(cfg.GetOpenAIAPIKey())
		transcriptionService = services.NewTranscriptionService(audioExtractorService, whisperClient, sseHub, slog.Default())
		slog.Info("Transcription service initialized (Whisper API enabled)")
	} else {
		transcriptionService = services.NewTranscriptionService(audioExtractorService, nil, sseHub, slog.Default())
		slog.Info("Transcription service initialized (disabled — missing OPENAI_API_KEY or FFmpeg)")
	}

	// Initialize AI terminology correction (Story 9.1)
	// Uses TextCompleter interface — provider created once here, not inside the service
	var terminologyService *services.TerminologyCorrectionService
	if cfg.HasClaudeKey() {
		terminologyProvider := ai.NewClaudeProvider(cfg.GetClaudeAPIKey())
		terminologyService = services.NewTerminologyCorrectionService(terminologyProvider)
	}
	if terminologyService != nil {
		subtitleEngine.SetTerminologyService(terminologyService)
		slog.Info("AI terminology correction enabled")
	}
	slog.Info("Subtitle engine initialized", "providers", len(subtitleProviders))

	// Initialize event emitter for real-time parse progress (Story 3.10)
	parseEventEmitter := events.NewChannelEmitter()
	defer parseEventEmitter.Close()

	// Initialize parse progress handler early so we can defer Close()
	parseProgressHandler := handlers.NewParseProgressHandler(parseEventEmitter)
	defer parseProgressHandler.Close()

	// Initialize handlers with injected service interfaces
	// Following Handler → Service → Repository → Database architecture
	movieHandler := handlers.NewMovieHandler(movieService)
	seriesHandler := handlers.NewSeriesHandler(seriesService)
	settingsHandler := handlers.NewSettingsHandler(settingsService)
	setupHandler := handlers.NewSetupHandler(setupService)
	mediaHandler := handlers.NewMediaHandler(mediaService)
	tmdbHandler := handlers.NewTMDbHandler(tmdbService)
	parserHandler := handlers.NewParserHandler(parserService)
	metadataHandler := handlers.NewMetadataHandler(metadataService)
	learningHandler := handlers.NewLearningHandler(learningService)
	retryHandler := handlers.NewRetryHandler(retryService)
	connectionHistoryService := services.NewConnectionHistoryService(repos.ConnectionHistory)
	serviceHealthHandler := handlers.NewServiceHealthHandler(degradationService)
	serviceHealthHandler.SetHistoryService(connectionHistoryService)
	qbittorrentHandler := handlers.NewQBittorrentHandler(qbittorrentService)
	downloadHandler := handlers.NewDownloadHandler(downloadService)
	libraryService := services.NewLibraryService(repos.Movies, repos.Series, repos.Episodes, services.WithTMDbVideos(tmdbService.VideosProvider()))
	libraryHandler := handlers.NewLibraryHandler(libraryService)
	mediaLibrariesHandler := handlers.NewMediaLibrariesHandler(mediaLibraryService)
	recentMediaHandler := handlers.NewRecentMediaHandler(movieService, seriesService)
	logHandler := handlers.NewLogHandler(logService)
	cacheHandler := handlers.NewCacheHandler(cacheStatsService, cacheCleanupService)
	serviceStatusService := services.NewServiceStatusService(healthMonitor, healthChecker)
	statusHandler := handlers.NewStatusHandler(serviceStatusService)
	backupHandler := handlers.NewBackupHandler(backupService)
	backupHandler.SetScheduler(backupScheduler)
	exportHandler := handlers.NewExportHandler(exportService)
	scannerHandler := handlers.NewScannerHandler(scannerService)
	scannerHandler.SetScheduler(scanScheduler)
	scannerHandler.SetEnrichmentService(enrichmentService)
	transcriptionHandler := handlers.NewTranscriptionHandler(movieService, transcriptionService)
	subtitleHandler := handlers.NewSubtitleHandler(
		subtitleProviders, subtitleScorer, subtitleConverter, subtitlePlacer,
		sseHub, repos.Movies, repos.Series,
	)
	// Wire batch processor (Story 8-9)
	batchCollector := subtitle.NewRepoCollector(repos.Movies, repos.Series, repos.Episodes)
	batchProcessor := subtitle.NewBatchProcessor(subtitleEngine, sseHub, batchCollector, subtitle.DefaultBatchConfig())
	subtitleHandler.SetBatchProcessor(batchProcessor)
	// parseProgressHandler already initialized above with defer Close()
	slog.Info("Handlers initialized with service injection")

	// Create Gin router
	router := gin.Default()

	// Security headers middleware (replaces Nginx security headers)
	router.Use(securityHeadersMiddleware())

	// Gzip compression middleware (replaces Nginx gzip)
	// Exclude SSE endpoint to preserve streaming (http.Flusher compatibility)
	router.Use(gzip.Gzip(gzip.DefaultCompression,
		gzip.WithExcludedPaths([]string{"/api/v1/events"}),
	))

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
		logHandler.RegisterRoutes(apiV1)    // Must be before settingsHandler to avoid /settings/:key conflict
		cacheHandler.RegisterRoutes(apiV1) // Must be before settingsHandler to avoid /settings/:key conflict
		statusHandler.RegisterRoutes(apiV1) // Must be before settingsHandler to avoid /settings/:key conflict
		backupHandler.RegisterRoutes(apiV1) // Must be before settingsHandler to avoid /settings/:key conflict
		exportHandler.RegisterRoutes(apiV1) // Must be before settingsHandler to avoid /settings/:key conflict
		settingsHandler.RegisterRoutes(apiV1)
		setupHandler.RegisterRoutes(apiV1)
		mediaHandler.RegisterRoutes(apiV1)
		tmdbHandler.RegisterRoutes(apiV1)
		parserHandler.RegisterRoutes(apiV1)
		metadataHandler.RegisterRoutes(apiV1)
		learningHandler.RegisterRoutes(apiV1)
		parseProgressHandler.RegisterRoutes(apiV1)
		handlers.RegisterRetryRoutes(apiV1, retryHandler)
		qbittorrentHandler.RegisterRoutes(apiV1)
		downloadHandler.RegisterRoutes(apiV1)
		libraryHandler.RegisterRoutes(apiV1)
		mediaLibrariesHandler.RegisterRoutes(apiV1) // /api/v1/libraries CRUD (Story 7b-2)
		recentMediaHandler.RegisterRoutes(apiV1)
		scannerHandler.RegisterRoutes(apiV1)
		subtitleHandler.RegisterRoutes(apiV1)
		transcriptionHandler.RegisterRoutes(apiV1)
		// SSE event stream endpoint
		apiV1.GET("/events", sse.Handler(sseHub))
		// Health services endpoint (Story 3.12 - Graceful Degradation)
		apiV1.GET("/health/services", serviceHealthHandler.GetServicesHealth)
		// Connection history endpoint (Story 4.6 - Connection Health Monitoring)
		apiV1.GET("/health/services/:service/history", serviceHealthHandler.GetConnectionHistory)
	}
	slog.Info("API routes registered", "prefix", "/api/v1")

	// Register static file serving and SPA fallback (must be AFTER all API routes)
	publicDir := getPublicDir()
	registerStaticRoutes(router, publicDir)
	slog.Info("Static file serving configured", "public_dir", publicDir)

	// Start retry scheduler for auto-retry mechanism (Story 3.11)
	if err := retryService.StartScheduler(ctx); err != nil {
		slog.Error("Failed to start retry scheduler", "error", err)
		// Non-fatal error - continue without retry scheduler
	} else {
		slog.Info("Retry scheduler started")
	}

	// Start backup scheduler (Story 6.8)
	schedulerCtx, schedulerCancel := context.WithCancel(context.Background())
	go backupScheduler.Start(schedulerCtx)
	slog.Info("Backup scheduler started")

	// Start scan scheduler (Story 7.2)
	scanSchedulerCtx, scanSchedulerCancel := context.WithCancel(context.Background())
	go scanScheduler.Start(scanSchedulerCtx)
	slog.Info("Scan scheduler started")

	// Start qBittorrent health monitoring with 30s interval (Story 4.6 - NFR-R6)
	monitorCtx, monitorCancel := context.WithCancel(context.Background())
	go healthMonitor.StartQBMonitoring(monitorCtx)
	slog.Info("qBittorrent health monitoring started (30s interval)")

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

	// Stop health monitoring goroutine
	slog.Info("Stopping health monitoring...")
	monitorCancel()

	// Stop scan scheduler (Story 7.2)
	slog.Info("Stopping scan scheduler...")
	scanSchedulerCancel()
	scanScheduler.Stop()

	// Stop backup scheduler
	slog.Info("Stopping backup scheduler...")
	schedulerCancel()
	backupScheduler.Stop()

	// Stop retry scheduler
	slog.Info("Stopping retry scheduler...")
	retryService.StopScheduler()

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
