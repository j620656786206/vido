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
	"github.com/vido/api/internal/cache"
	"github.com/vido/api/internal/config"
	"github.com/vido/api/internal/crypto"
	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/events"
	"github.com/vido/api/internal/handlers"
	"github.com/vido/api/internal/health"
	"github.com/vido/api/internal/images"
	"github.com/vido/api/internal/logger"
	"github.com/vido/api/internal/plugins"
	"github.com/vido/api/internal/plugins/radarr"
	"github.com/vido/api/internal/plugins/sonarr"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/retry"
	"github.com/vido/api/internal/secrets"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/sse"
	"github.com/vido/api/internal/subtitle"
	subtitleproviders "github.com/vido/api/internal/subtitle/providers"
	// Media config is loaded during service initialization
	// and validates directories from VIDO_MEDIA_DIRS env var
	//
	// NOTE: migrations register via init() through the NAMED import above —
	// the former duplicate blank import was pre-existing ST1019 (staticcheck),
	// removed by sub-2-2a as a quick in-place fix.
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

	// bugfix-system-logs-no-retention: prune expired system_logs rows once at
	// startup, then reclaim file space while nothing else contends the writer
	// lock — before the HTTP listener and schedulers start is the only
	// uncontended window for a VACUUM. Ongoing pruning rides the cache-sweep
	// scheduler below.
	if cfg.LogRetentionDays > 0 {
		if removed, err := logRepo.DeleteOlderThan(ctx, cfg.LogRetentionDays); err != nil {
			slog.Warn("Startup system_logs prune failed", "error", err)
		} else if removed > 0 {
			slog.Info("Startup system_logs prune complete",
				"removed", removed, "retention_days", cfg.LogRetentionDays)
		}
	}
	if _, err := db.ReclaimSpaceIfBloated(ctx); err != nil {
		slog.Warn("Database space reclaim failed", "error", err)
	}

	// Database supervisor (bugfix-i-1): background liveness watch that captures
	// evidence and recycles a wedged connection pool instead of leaving the
	// process permanently unhealthy until a reinstall. Its Healthy() verdict
	// also feeds the /api/v1 DatabaseGate (bugfix-i-3). Started with the other
	// background loops near the end of main.
	dbSupervisor := database.NewSupervisor(db)

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
	availabilityService := services.NewAvailabilityService(repos.Movies, repos.Series) // Story 10-4
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

	// Story 12-2: wire the season/episode accordion deps into the series service
	// (episode repo for local subtitle/file status + TMDb for the canonical episode list).
	seriesService.SetEpisodeDeps(repos.Episodes, tmdbService)
	seriesService.SetSeasonRepo(repos.Seasons) // bugfix-20-1: GetSeasons reads the seasons table

	// Initialize explore block service (Story 10.3 — homepage custom discover blocks)
	exploreBlockService := services.NewExploreBlockService(repos.ExploreBlocks, tmdbService, repos.Cache)
	if err := exploreBlockService.SeedDefaultsIfEmpty(context.Background()); err != nil {
		slog.Warn("Failed to seed default explore blocks", "error", err)
	}

	// Initialize filter preset service (Story 11.4 — saved discover filter presets)
	filterPresetService := services.NewFilterPresetService(repos.FilterPresets)

	// Initialize request service (Story 13-1a — Epic 13 one-click 想要 requests).
	// Intent-only: fulfilment lands in 13-4, status transitions/SSE in 13-3a.
	// 13-2a: the episode repo backs the episode-level owned guard + coverage.
	requestService := services.NewRequestService(repos.Requests, tmdbService, repos.Movies, repos.Series, repos.Episodes)

	// Story 13-4a — *arr DVR plugin infrastructure (§7). The manager owns
	// per-plugin config (settings + secrets), fingerprint-cached clients, and
	// the 60s health scheduler (self-contained — the 5-service ServicesHealth
	// model is deliberately not extended). Movie fulfilment rides the
	// request-create path via the optional nil-safe dependency.
	pluginManager := plugins.NewManager(repos.Settings, secretsService, repos.ConnectionHistory, slog.Default(), 0)
	pluginManager.Register("radarr", func(config plugins.PluginConfig) plugins.DVRPlugin {
		return radarr.NewClient(config)
	})
	// Story 13-4b — Sonarr rides the same manager/scheduler/settings. The
	// TVDB resolver closes over the shared TMDb service (Rule 27 reuse:
	// shared limiter/cache/key; the sonarr package stays services-free).
	pluginManager.Register("sonarr", func(config plugins.PluginConfig) plugins.DVRPlugin {
		return sonarr.NewClient(config, sonarr.TVDBResolverFunc(
			func(ctx context.Context, tmdbID int64) (int64, error) {
				ids, err := tmdbService.GetTVExternalIDs(ctx, int(tmdbID))
				if err != nil {
					return 0, err
				}
				return ids.TVDbID, nil
			}))
	})
	fulfilmentService := services.NewFulfilmentService(pluginManager, repos.Settings, repos.Requests)
	requestService.SetFulfilmentService(fulfilmentService)
	dvrSettingsService := services.NewDVRSettingsService(pluginManager, repos.Settings, secretsService)

	// Shared AI throttle (Story 9R-11): one Governor caps concurrency + QPS
	// across ASR (Whisper), LLM (Claude) AND the parse-path providers below
	// (sub-5-1 CR H2) so a library-wide scan or batch can't fan out unbounded
	// requests. Created here — before the parse-path AI service — and injected
	// into every client construction that follows.
	aiGovernor := ai.NewGovernor(cfg.AIMaxConcurrent, cfg.AIRatePerSec, cfg.AIMaxConcurrent)
	slog.Info("AI governor initialized", "max_concurrent", cfg.AIMaxConcurrent, "rate_per_sec", cfg.AIRatePerSec, "run_budget_usd", cfg.AIRunBudgetUSD)

	// Initialize AI service for AI-powered filename parsing (Story 3.1)
	aiService, err := services.NewAIService(cfg, db.Conn(), aiGovernor)
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
	scannerService.SetEpisodeRepo(repos.Episodes)       // Story 9c-3: series file_size aggregation

	// TV routing (bugfix-b): without this the scanner writes every scanned file to `movies`,
	// which is what left series/seasons/episodes empty while the movie table filled up with
	// one row per episode.
	mediaIngestService := services.NewMediaIngestService(repos.Series, repos.Seasons, repos.Episodes, slog.Default())
	scannerService.SetTVIngest(mediaIngestService, parserService)
	slog.Info("Scanner service initialized (TV routing enabled)")

	// Initialize NFO reader service for .nfo sidecar parsing (Story 9c-2)
	nfoReaderService := services.NewNFOReaderService(slog.Default())
	slog.Info("NFO reader service initialized")

	// Initialize FFprobe service for video technical info extraction (Story 9c-3)
	ffprobeService := services.NewFFprobeService(3, 10*time.Second, slog.Default())
	// sub-6-3: ONE extraction gate for the whole process (Rule 14) — every
	// Extractor below shares it, so two workers never demux the same disk at
	// once — and the configured extraction floor; the per-file bound grows
	// with size inside the Extractor.
	subtitleExtractGate := subtitle.NewExtractGate()
	subtitleExtractTimeout := time.Duration(cfg.SubtitleExtractTimeoutSeconds) * time.Second
	subtitleExtractPerGB := time.Duration(cfg.SubtitleExtractPerGBSeconds) * time.Second
	subtitleExtractorOpts := []subtitle.ExtractorOption{
		subtitle.WithExtractGate(subtitleExtractGate),
		subtitle.WithPerGBTimeout(subtitleExtractPerGB),
	}
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
	// bugfix-b: the scanner now creates series rows with a folder-name title and
	// parse_status=pending. Without a series repo the enrichment pass would leave them
	// unmatched forever.
	enrichmentService.SetSeriesRepo(repos.Series)

	// Wire post-scan auto-enrichment: after scan completes with new/updated files,
	// automatically trigger metadata enrichment in background.
	//
	// sub-1-6 AC #2: SetOnScanComplete holds exactly ONE callback, so the FR13
	// subtitle-pipeline enqueue must WRAP this body, never call the setter a
	// second time. The body is hoisted into a variable so the composition below
	// (subtitle.ComposeScanCallback) can preserve it byte-for-byte.
	postScanEnrichment := func() {
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
	}
	scannerService.SetOnScanComplete(postScanEnrichment)
	slog.Info("Enrichment service initialized with post-scan auto-trigger")

	// Initialize scan scheduler (Story 7.2)
	scanScheduler := services.NewScanScheduler(scannerService, repos.Settings, slog.Default())
	slog.Info("Scan scheduler initialized")

	// Initialize cache expiry sweep scheduler — sweeps up to 3 DB-table caches (cache_entries +
	// offline_cache + ai_cache) on one shared ticker (infra-cache-entries-expiry-sweep +
	// infra-ai-offline-cache-expiry-sweep). offline_cache is always constructed (:124); the ai_cache
	// target is added only when an AI provider is configured (aiService is nil otherwise, :214).
	cacheSweepExtra := []services.CacheSweepTarget{
		services.SweepTarget("offline_cache", offlineCache),
	}
	if aiService != nil {
		cacheSweepExtra = append(cacheSweepExtra, services.SweepFunc("ai_cache", aiService.ClearExpiredCache))
	}
	// system_logs retention rides the same sweep ticker
	// (bugfix-system-logs-no-retention); the startup prune above handled the
	// backlog, this keeps steady-state growth bounded.
	if cfg.LogRetentionDays > 0 {
		retentionDays := cfg.LogRetentionDays
		cacheSweepExtra = append(cacheSweepExtra, services.SweepFunc("system_logs_retention",
			func(ctx context.Context) (int64, error) {
				return logRepo.DeleteOlderThan(ctx, retentionDays)
			}))
	}
	cacheSweepScheduler := services.NewCacheSweepScheduler(repos.Cache, repos.Settings, cacheSweepExtra...)
	slog.Info("Cache sweep scheduler initialized")

	// Initialize download progress broadcaster (ux3-4-2b — Epic 14 H-1 / P3-012). One gated
	// server-side qBittorrent poll → SSE fan-out replaces N-browser polling of GET /downloads. Gated on
	// sseHub.ClientCount(): zero clients on the Downloads page → zero qBittorrent traffic.
	downloadProgressBroadcaster := services.NewDownloadProgressBroadcaster(downloadService, sseHub)
	slog.Info("Download progress broadcaster initialized")

	// Story 13-3a — the always-on request reconciler (Epic 13 G-3/P3-003):
	// derives each active request's status from library ownership + *arr
	// queues + qBT every 15s, persists transitions, triggers the debounced
	// import-window scan, retries stranded pending rows, and (client-gated)
	// broadcasts the request_progress SSE snapshot. The 13-5 subtitle trigger
	// plugs into its OnRequestCompleted seam later — nothing wired here.
	requestStatusPoller := services.NewRequestStatusPoller(
		repos.Requests, availabilityService, pluginManager, downloadService,
		scannerService, fulfilmentService, sseHub,
	)
	// 13-2a (CR H1): a partial request lives on a show that is ALREADY local,
	// so the poller's title-level completion rule needs the episode-level
	// refinement the request service already computes.
	requestStatusPoller.SetSelectionOwnershipChecker(requestService)
	slog.Info("Request status poller initialized")

	// Library type-change rebuild (bugfix-library-type-change-no-reclassify):
	// changing a library's content type purges its rows and rescans — the
	// automated delete-and-recreate Plex/Jellyfin make you do by hand.
	if mp, ok := repos.Movies.(*repository.MovieRepository); ok {
		if sp, ok2 := repos.Series.(*repository.SeriesRepository); ok2 {
			mediaLibraryService.SetMediaPurgers(mp, sp)
		}
	}
	mediaLibraryService.SetScanTrigger(func() {
		go func() {
			if _, err := scannerService.StartScan(context.Background()); err != nil {
				slog.Error("Post-type-change rescan failed", "error", err)
			}
		}()
	})

	// Initialize subtitle engine components (Story 8.1-8.8)
	subtitleConverter, _ := subtitle.NewConverter()
	subtitleScorer := subtitle.NewScorer(subtitle.NewDefaultScorerConfig())
	subtitlePlacer := subtitle.NewPlacer(subtitle.DefaultPlacerConfig())
	// Initialize subtitle providers (Assrt, OpenSubtitles).
	// Zimuku removed 2026-07-05 (9R-14): zimuku.org sits behind a Yunsuo anti-bot
	// WAF — every query returns ErrCaptchaDetected (ADR route-c Decision 1/D3).
	assrtProvider := subtitleproviders.NewAssrtProvider(ctx, secretsService)
	opensubProvider := subtitleproviders.NewOpenSubProvider(ctx, secretsService)
	subtitleProviders := []subtitleproviders.SubtitleProvider{assrtProvider, opensubProvider}
	subtitleEngine := subtitle.NewEngine(
		subtitleProviders, subtitleScorer, subtitleConverter, subtitlePlacer,
		sseHub, repos.Movies, repos.Series,
	)
	// Story 13-5 (artery #5): request completed → automatic subtitle search for
	// the media that just landed, via the poller's OnRequestCompleted seam.
	// Fires once per transition edge; the trigger itself skips media that
	// already has a subtitle outcome, and its failures never touch the request.
	requestSubtitleTrigger := subtitle.NewRequestCompletionTrigger(repos.Movies, repos.Series, subtitleEngine)
	requestStatusPoller.OnRequestCompleted = requestSubtitleTrigger.OnRequestCompleted
	slog.Info("Request-completion subtitle trigger wired (13-5)")
	// aiGovernor was created before the parse-path AI service (sub-5-1 CR H2)
	// — the same instance throttles the Whisper + Claude clients below.

	// Initialize audio extractor service (Story 9.2a)
	// sub-6-3: ASR audio extraction is ffmpeg on the same disk, so it takes
	// turns with subtitle extraction through the same gate.
	audioExtractorService := services.NewAudioExtractorService(1, 5*time.Minute, slog.Default(),
		services.WithAudioExtractSlot(extractSlotAdapter{gate: subtitleExtractGate}))
	slog.Info("Audio extractor service initialized", "available", audioExtractorService.IsAvailable())

	// ── Provider keys: resolver + hot-reloadable holders (sub-2-1a AC #1/#2,
	//    extended to ASR by sub-5-2 AC #1/#2) ──────────────────────────────────
	//
	// The key is resolved secret-first so a key typed into the settings page
	// actually reaches the pipeline; it used to be env-only, which made the page
	// a silent no-op (Break 1). The holders rebuild their client when the resolved
	// key changes, so a runtime edit takes effect without a restart (Break 2).
	//
	// Declared HERE (moved up by sub-5-2) because the transcription service below
	// now takes the ASR holder instead of a boot-built client.
	keyResolver := services.NewKeyResolver(secretsService, services.EnvKeys{
		Claude: cfg.GetClaudeAPIKey(),
		TMDb:   cfg.GetTMDbAPIKey(),
		OpenAI: cfg.GetOpenAIAPIKey(),
	}, slog.Default())
	claudeHolder := services.NewClaudeProviderHolder(
		keyResolver, cfg.GetClaudeModel(), slog.Default(), ai.WithClaudeGovernor(aiGovernor))

	// Initialize ASR provider holder and transcription service (Story 9.2a;
	// hot-reload by sub-5-2). Constructed UNCONDITIONALLY: the old
	// `if cfg.HasOpenAIKey()` guard froze the boot-time key into EXISTENCE, so a
	// key saved later in /settings/keys reached a client that was never built —
	// and the else-branch skipped the four pipeline setters below, which meant a
	// keyless boot would have produced a silently DEGRADED transcription run
	// (no budget ceiling, no per-show glossary, no OpenCC safety net, no atomic
	// placer) even after the key arrived. Availability is now a per-call question
	// the service asks the holder (TranscriptionService.IsAvailable).
	asrHolder := services.NewASRProviderHolder(
		keyResolver, cfg.ASRBaseURL, cfg.ASRModel, slog.Default(), ai.WithWhisperGovernor(aiGovernor))
	transcriptionService := services.NewTranscriptionService(audioExtractorService, asrHolder, sseHub, slog.Default())
	transcriptionService.SetRunBudgetUSD(cfg.AIRunBudgetUSD)
	// 9R-10: wire the per-show glossary + OpenCC safety net + atomic placer
	// into the Route C generation pipeline.
	transcriptionService.SetGlossaryRepository(repos.Glossary)
	if subtitleConverter != nil {
		transcriptionService.SetOpenCCConverter(subtitleConverter)
	}
	transcriptionService.SetPlacer(subtitlePlacerAdapter{subtitlePlacer})
	slog.Info("Transcription service initialized",
		"ffmpeg_available", audioExtractorService.IsAvailable(),
		"asr_configured", asrHolder.IsConfigured(ctx),
		"base_url_override", cfg.ASRBaseURL != "",
		"model_override", cfg.ASRModel != "",
		"available", transcriptionService.IsAvailable())
	// 9R-16 AC 12: persist generation success (subtitle_status/path/language)
	// so the missing-scope batch enumeration shrinks and poster badges flip.
	transcriptionService.SetSubtitleStatusWriter(repos.Movies)
	// sub-2-2a AC #3: row-state read behind the translate-only resume — an
	// `untranslated` row with its EN SRT still on disk skips extract+ASR.
	transcriptionService.SetSubtitleStateReader(repos.Movies)
	// sub-3-2: the EPISODE writer/reader pair behind WithMediaType dispatch —
	// the pipeline's ASR fallback leg serves episodes through the same run.
	transcriptionService.SetEpisodeSubtitleStatusWriter(repos.Episodes)
	transcriptionService.SetEpisodeSubtitleStateReader(repos.Episodes)
	// 9R-8: the parent-series row behind an episode's FR26 prompt context. The
	// MOVIE half needs no wiring — SetSubtitleStateReader above already hands
	// the service the complete movie row.
	transcriptionService.SetSeriesMetadataReader(repos.Series)

	// Initialize AI terminology correction (Story 9.1) + subtitle translation (Story 9.2b).
	// Constructed UNCONDITIONALLY (sub-2-1a AC #2): they take the holder, which
	// declines with ErrAINotConfigured while no key resolves and starts working
	// the moment one is saved. The old `if cfg.HasClaudeKey()` guard left these
	// nil forever on a keyless boot, so a key added later reached nothing.
	terminologyService := services.NewTerminologyCorrectionService(claudeHolder)
	translationService := services.NewTranslationService(claudeHolder, sseHub)
	subtitleEngine.SetTerminologyService(terminologyService)
	transcriptionService.SetTranslationService(translationService)
	slog.Info("AI services wired through the key holder",
		"claude_configured", claudeHolder.IsConfigured(ctx))
	slog.Info("Subtitle engine initialized", "providers", len(subtitleProviders))

	// ── Subtitle generation pipeline (sub-1-6: D5 flag seam + FR13 + FR23) ──
	//
	// The whole M1 generation path hangs off ONE env var. `legacy` (the default)
	// leaves every variable below nil, which is exactly what the batch seam,
	// the scan callback and the endpoint each read as "stay on the shipped
	// path" — the flag is never re-read anywhere downstream (D5's ban).
	var (
		subtitlePipeline     *subtitle.Pipeline
		subtitlePipelinePool *subtitle.WorkerPool
		// Hoisted so the graceful-shutdown block can Stop() it
		// (bugfix-autogenerator-no-timeout-or-shutdown AC #6). nil in legacy mode.
		autoGenerator *subtitle.AutoGenerator
	)
	// Built unconditionally: the FR12 endpoint uses it to answer 404 for an
	// unknown media id, and it is three struct fields — nothing is started.
	subtitlePipelineMedia := subtitle.NewMediaStore(repos.Movies, repos.Series, repos.Episodes)
	// AC #5: ONE capability predicate, read by all THREE entry points — the
	// endpoint (409), the scanner enqueue sweep, and the batch seam. Declared
	// here so there is exactly one definition of "the pipeline can run".
	// sub-2-1a AC #5 re-point: the gate now asks the RESOLVER, not the boot-time
	// env snapshot, so saving a key in the settings page un-gates the pipeline
	// without a restart. Still a plain func() bool — no Rule 20 bump owed.
	subtitleCapabilityGate := func() bool { return keyResolver.Has(context.Background(), services.KeyClaude) }
	// CR sub-2-1a H1: construction is gated ONLY by the mode flag. Keeping
	// `&& subtitleCapabilityGate()` here froze the gate's boot-time value into
	// EXISTENCE — a keyless boot never built the pool, so a key saved later from
	// the settings page un-gated an endpoint whose queue was still nil (409 until
	// restart, the exact class this story removes). The gate still governs every
	// RUNTIME entry point: the endpoint's 409, the EnqueueMissing scan sweep, and
	// the per-batch seam below.
	if cfg.SubtitlePipelineEnabled() {
		subtitleExtractor := subtitle.NewExtractor(subtitleExtractTimeout, slog.Default(), subtitleExtractorOpts...)
		subtitleRouter := subtitle.NewRouter(
			ffprobeService,
			subtitleExtractor,
			slog.Default(),
		)
		// sub-3-1: the ASR fallback port + the sweep's availability gate share
		// one adapter over the SAME transcription service the manual Route-C
		// dialog uses, so a no_text_source movie is recovered by exactly the
		// pipeline 生成字幕 would run. Wired unconditionally in pipeline mode —
		// an ASR-less boot degrades per item via the service's own entry gate.
		pipelineASR := pipelineASRAdapter{ts: transcriptionService}
		subtitlePipeline = subtitle.NewPipeline(
			translationService, subtitleConverter, slog.Default(),
			subtitle.WithRouter(subtitleRouter),
			subtitle.WithPlacer(subtitlePlacer),
			subtitle.WithMediaStore(subtitlePipelineMedia),
			subtitle.WithRunStore(repos.SubtitleRuns),
			subtitle.WithSegmentCache(subtitle.NewSegmentCacheRepository(repos.Cache)),
			// sub-5-5: per-show glossary feed (去程) + auto-harvest write-back
			// (回程) over the SAME show_glossary table the legacy path and the
			// 9R-15 review REST already use.
			subtitle.WithGlossaryStore(subtitle.NewGlossaryStoreRepository(repos.Glossary)),
			// sub-6-5: the model id comes from the holder that actually sends
			// the request, not from the env override — which was "" on every
			// default-model run and left subtitle_runs.model_id empty.
			subtitle.WithModelSource(claudeHolder.EffectiveModel),
			// sub-5-1 AC #3: per-item AI cost ceiling for the FR12/pool path —
			// a ctx already carrying a Budget (the sub-4-2 consent batch) keeps
			// its shared ceiling; only budget-less entries get this envelope.
			subtitle.WithRunBudgetUSD(cfg.AIRunBudgetUSD),
			subtitle.WithSpeechTranscriber(pipelineASR),
			// AC #6: FR33/P8 progress. Same event type and payload shape the
			// search path already broadcasts — sse/hub.go stays untouched.
			subtitle.WithProgress(subtitle.NewSSEProgressHook(sseHub)),
		)
		subtitlePipelinePool = subtitle.NewWorkerPool(subtitlePipeline, slog.Default(),
			subtitle.WithCandidateFinders(repos.Movies, repos.Episodes),
			subtitle.WithCapabilityGate(subtitleCapabilityGate),
			subtitle.WithASRAvailability(pipelineASR.Available),
		)
		// sub-4-1 AC #1: a completed scan does NOT enqueue subtitle generation.
		// sub-1-6 AC #2 (FR13) wired the library-wide sweep here; the first
		// production run showed why that is wrong — one press of 掃描媒體庫
		// enqueued 1026 items, ~2/3 of them onto PAID speech recognition, with
		// no number ever shown to the user. Generation is now chosen explicitly
		// on a screen that shows the estimate first (sub-4-3).
		//
		// The pool below is deliberately KEPT: the manual per-item endpoint
		// (FR12) still drives it. The sub-4-2 consented batch does NOT go
		// through the pool — it calls Pipeline.ProcessItem directly (the pool's
		// process-wide queue has no batch identity, no removal for cancel, and
		// no per-batch budget ctx); see pipelineGenerationRunner.
		// `internal/cost_consent_test.go` fails if a sweep caller reappears.
		//
		// 9R-10b: the scan-complete slot is COMPOSED, never re-registered.
		// `postScanEnrichment` above is passed through byte-for-byte and runs
		// FIRST. Ordering is convention, NOT a dependency (補審 correction):
		// SelectAndRoute probes the FILE, and neither free route reads
		// item.Context at all — enrichment metadata reaches the pipeline only
		// via RouteTranslate and the runVersion hash. The two callbacks are
		// each `go`-spawned, so they overlap regardless; what makes the overlap
		// safe is enrichment's NARROW writer (UpdateEnrichedMetadata), not this
		// order. What follows it is the FREE lane only: every item
		// is processed with ProcessItemOptions.FreeOnly, so an embedded Chinese
		// track is finished locally while anything that would bill stops at the
		// threshold and waits for the estimate screen. The paid sweep this slot
		// once held stays gone.
		autoGenerator = subtitle.NewAutoGenerator(
			subtitlePipeline,
			autoSubtitlePolicyAdapter{libraries: repos.MediaLibraries},
			slog.Default(),
			subtitle.WithAutoCandidateFinders(repos.Movies, repos.Episodes),
			subtitle.WithAutoSeriesResolver(repos.Series),
			// CR H1: lets the trigger skip items it has already parked awaiting
			// consent, so the per-run budget moves down the list instead of
			// re-extracting the same paid items on every scan.
			subtitle.WithAutoDeferredRuns(repos.SubtitleRuns),
			// sub-6-3: the free lane's per-item deadline follows the
			// extractor's size-aware bound, or a 93 GB file would get a
			// 46-minute ffmpeg deadline under a 15-minute item deadline.
			subtitle.WithAutoExtractTimeout(func(path string) time.Duration {
				bound, _ := subtitleExtractor.EffectiveTimeout(path)
				return bound
			}),
		)
		scannerService.SetOnScanComplete(
			subtitle.ComposeScanCallback(postScanEnrichment, autoGenerator.ScanCallback()),
		)
		slog.Info("Subtitle generation pipeline enabled",
			"mode", cfg.SubtitlePipelineMode, "workers", subtitle.PipelineConcurrencyM1, "model", claudeHolder.EffectiveModel(),
			// May be false on a keyless boot: the pool exists and idles behind the
			// gate, and starts accepting work the moment a key is saved (sub-2-1a).
			"translation_configured", subtitleCapabilityGate(),
			// Loud on purpose: the behaviour change is the whole point of sub-4-1.
			"scan_auto_enqueue", false,
			// 9R-10b: free lane only, per-library opt-in, default OFF.
			"scan_auto_free_generation", true,
			"scan_auto_item_timeout", subtitle.AutoGenerationItemTimeout,
			"extract_timeout_floor", subtitleExtractTimeout,
			"extract_per_gb", subtitleExtractPerGB,
			"auto_max_per_run", subtitle.AutoGenerationMaxPerRun)
	} else {
		// ONE line, at wiring time — not one per scanned item (AC #5).
		slog.Info("Subtitle generation pipeline disabled — staying on the legacy search path",
			"mode", cfg.SubtitlePipelineMode, "translation_configured", subtitleCapabilityGate())
	}

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
	// Story 12-1 — dual rating display. Reuses the SAME Douban provider that the
	// metadata service owns (single rate limiter / circuit breaker for douban.com).
	// When Douban is disabled DoubanProvider() returns a nil *DoubanProvider; keep
	// the searcher a genuine nil interface (avoid the typed-nil-in-interface trap)
	// so the service takes its graceful-degradation path instead of dereferencing.
	// Story 12-6 — Douban review summary. The SAME *DoubanProvider also satisfies
	// services.DoubanReviewScraper (cache-aware ScrapeReviewSummary), so the review
	// scrape rides the existing client / limiter / cache (Rule 27 ①/②). Keep it a
	// genuine nil interface when Douban is disabled (typed-nil-in-interface trap).
	var doubanSearcher services.DoubanSearcher
	var doubanReviewScraper services.DoubanReviewScraper
	if dp := metadataService.DoubanProvider(); dp != nil {
		doubanSearcher = dp
		doubanReviewScraper = dp
	}
	doubanRatingService := services.NewDoubanRatingService(doubanSearcher, doubanReviewScraper, repos.Movies, repos.Series)
	doubanRatingHandler := handlers.NewDoubanRatingHandler(doubanRatingService)
	settingsHandler := handlers.NewSettingsHandler(settingsService)
	setupHandler := handlers.NewSetupHandler(setupService)
	mediaHandler := handlers.NewMediaHandler(mediaService)
	availabilityHandler := handlers.NewAvailabilityHandler(availabilityService) // Story 10-4
	tmdbHandler := handlers.NewTMDbHandler(tmdbService)
	// Story 12-3 — related-content recommendations (TMDb recs/similar + ownership join).
	recommendationService := services.NewRecommendationService(tmdbService, repos.Movies, repos.Series)
	tmdbHandler.SetRecommendationService(recommendationService)
	// Story 11-3 — unified dual-language instant search. SearchClient() returns nil
	// if the underlying TMDb client does not satisfy SearchTMDbClient (e.g. a future
	// caching decorator missing the *WithLanguage methods); fail fast at startup
	// rather than panicking on the first /api/v1/search request.
	searchClient := tmdbService.SearchClient()
	if searchClient == nil {
		slog.Error("Unified search client unavailable — TMDb client does not satisfy SearchTMDbClient")
		os.Exit(1)
	}
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
	// Unified search takes the library service as its local leg — owned items
	// stay searchable when TMDb is unreachable (testsprite-round1 TC092).
	searchService := services.NewSearchService(searchClient, libraryService)
	searchHandler := handlers.NewSearchHandler(searchService)
	libraryHandler := handlers.NewLibraryHandler(libraryService)
	// 補審 M4: the opt-in checkbox is only offered where the trigger that
	// honours it is actually built — the `if cfg.SubtitlePipelineEnabled()`
	// block above. The default mode is `legacy`, where it would be a promise
	// nothing keeps.
	mediaLibrariesHandler := handlers.NewMediaLibrariesHandler(
		mediaLibraryService,
		handlers.WithAutoSubtitleSupport(cfg.SubtitlePipelineEnabled),
	)
	exploreBlocksHandler := handlers.NewExploreBlocksHandler(exploreBlockService)                // Story 10.3
	filterPresetsHandler := handlers.NewFilterPresetsHandler(filterPresetService)                // Story 11.4
	requestHandler := handlers.NewRequestHandler(requestService)                                 // Story 13-1a
	glossaryHandler := handlers.NewGlossaryHandler(services.NewGlossaryService(repos.Glossary))  // Story 9R-15
	dvrSettingsHandler := handlers.NewDVRSettingsHandler(dvrSettingsService, "radarr", "sonarr") // Story 13-4a + 13-4b
	recentMediaHandler := handlers.NewRecentMediaHandler(movieService, seriesService)
	logHandler := handlers.NewLogHandler(logService)
	cacheHandler := handlers.NewCacheHandler(cacheStatsService, cacheCleanupService)
	serviceStatusService := services.NewServiceStatusService(healthMonitor, healthChecker)
	statusHandler := handlers.NewStatusHandler(serviceStatusService)
	statusSummaryService := services.NewStatusSummaryService(serviceStatusService, scannerService, downloadService, mediaLibraryService)
	statusSummaryHandler := handlers.NewStatusSummaryHandler(statusSummaryService)
	backupHandler := handlers.NewBackupHandler(backupService)
	backupHandler.SetScheduler(backupScheduler)
	exportHandler := handlers.NewExportHandler(exportService)
	scannerHandler := handlers.NewScannerHandler(scannerService)
	scannerHandler.SetScheduler(scanScheduler)
	scannerHandler.SetEnrichmentService(enrichmentService)
	// 9R-10a: repos.Episodes wires the per-episode transcribe route
	// (POST /episodes/:id/transcribe). Passing it is what mounts that route.
	transcriptionHandler := handlers.NewTranscriptionHandler(movieService, repos.Episodes, transcriptionService)

	// 9R-13: .nfo metadata localizer (movies) — additive zh-TW .nfo via the
	// shared translation + glossary infra. nil when no translation provider.
	nfoLocalizer := services.NewNFOLocalizerService(translationService, repos.Glossary, slog.Default())
	if nfoLocalizer != nil {
		// 9R-13a: episode enumeration behind ?include_episodes=true.
		nfoLocalizer.SetEpisodeLister(repos.Episodes)
	}
	nfoLocalizerHandler := handlers.NewNFOLocalizerHandler(movieService, seriesService, repos.Episodes, nfoLocalizer)
	subtitleHandler := handlers.NewSubtitleHandler(
		subtitleProviders, subtitleScorer, subtitleConverter, subtitlePlacer,
		sseHub, repos.Movies, repos.Series,
	)
	// Wire batch processor (Story 8-9)
	batchCollector := subtitle.NewRepoCollector(repos.Movies, repos.Series, repos.Episodes)
	// sub-1-6 AC #1: the D5 seam. A nil ItemProcessor IS legacy mode, so in
	// `legacy` this NewBatchProcessor call is byte-identical to the shipped one.
	batchOpts := []subtitle.BatchProcessorOption{}
	if subtitlePipeline != nil {
		batchOpts = append(batchOpts,
			subtitle.WithItemProcessor(subtitlePipeline),
			// AC #5's third entry point — the same predicate, re-read per batch.
			subtitle.WithPipelineGate(subtitleCapabilityGate))
	}
	batchProcessor := subtitle.NewBatchProcessor(subtitleEngine, sseHub, batchCollector, subtitle.DefaultBatchConfig(), batchOpts...)
	subtitleHandler.SetBatchProcessor(batchProcessor)
	// Consented generation batch (9R-16 orchestrator, sub-4-2 engine selection):
	// sequential single-flight over ONE shared AI budget, independent from the
	// fetch batchProcessor above (they share no state). Engine per mode:
	// pipeline mode drives the D2 pipeline directly (extract→translate free
	// route first, ASR only on no_text_source — the F15 quote must match what
	// runs); legacy mode keeps the Route C transcription engine, now with the
	// item's media type forwarded so episodes write back to the episodes table.
	var generationRunner services.GenerationRunner = services.NewRouteCGenerationRunner(transcriptionService)
	if subtitlePipeline != nil {
		generationRunner = pipelineGenerationRunner{
			pipeline: subtitlePipeline,
			// Share the pool's in-flight set so the batch and the FR12-driven
			// workers never process the same media concurrently (CR M4).
			guard:     subtitlePipelinePool,
			available: subtitleCapabilityGate,
		}
	}
	generationBatchProcessor := services.NewGenerationBatchProcessor(
		generationRunner, repos.Movies, repos.Episodes, sseHub, cfg.AIRunBudgetUSD, slog.Default())
	generationBatchHandler := handlers.NewGenerationBatchHandler(generationBatchProcessor)

	// Cost preview (story sub-4-1): what would generating subtitles cost, per
	// item and in total, WITHOUT spending anything. Registered in every mode —
	// it only reads and probes, so it is safe on a legacy install and lets the
	// UI answer "why is this costly?" before the pipeline is ever enabled.
	//
	// A prediction-only Router is built here rather than reusing the one inside
	// the pipeline-mode block above: NewRouter is stateless and cheap, and
	// PredictRoute never touches the extractor, so sharing scope would buy
	// nothing and couple this endpoint to the feature flag.
	// The self-hosted flag comes from ASR_BASE_URL via the SAME predicate the
	// Whisper client's metering uses (sub-5-1 CR M1) — a self-hosted endpoint
	// has no per-minute price, and quote vs invoice must come from one answer.
	generationCandidateService := services.NewGenerationCandidateService(
		repos.Movies, repos.Episodes,
		// sub-5-3 AC #1: the F15 group headers' series titles — one memoized
		// lookup per series per sweep, nil-safe fail-soft inside the service.
		repos.Series,
		routePredictorAdapter{router: subtitle.NewRouter(
			ffprobeService,
			subtitle.NewExtractor(subtitleExtractTimeout, slog.Default(), subtitleExtractorOpts...),
			slog.Default())},
		ai.IsSelfHostedASRBaseURL(cfg.ASRBaseURL),
		// sub-5-1 AC #5: the F15 prefill source — the envelope carries the
		// operator's real default instead of a frontend constant.
		cfg.AIRunBudgetUSD,
		slog.Default(),
	)
	generationCandidateService.SetSSEHub(sseHub)
	// sub-5-4: remember route verdicts against file identity (size + mtime), so
	// a repeat sweep only ffprobes what is new or actually changed. Rides the
	// existing cache_entries table — no migration, and the shared expiry sweep
	// above already covers the new `subtitle_route` family.
	generationCandidateService.SetRouteCache(services.NewRouteCacheRepository(repos.Cache))
	generationCandidatesHandler := handlers.NewGenerationCandidatesHandler(generationCandidateService)
	// FR12 manual trigger (sub-1-6 AC #4). The route is registered in EVERY
	// mode so the API surface does not change shape with an env var; a nil
	// queue (legacy) answers 409 rather than 404.
	var subtitlePipelineQueue handlers.SubtitlePipelineQueue
	if subtitlePipelinePool != nil {
		subtitlePipelineQueue = subtitlePipelinePool
	}
	// FR25 provider-key settings (sub-2-1a AC #3). Writable requires an
	// encryption key — without it the page renders read-only rather than
	// accepting input that would fail at the storage layer (AC #4).
	keySettingsService := services.NewKeySettingsService(keyResolver, secretsService, cfg.HasEncryptionKey())
	keySettingsHandler := handlers.NewKeySettingsHandler(keySettingsService, claudeHolder)
	subtitlePipelineHandler := handlers.NewSubtitlePipelineHandler(
		subtitlePipelineQueue, subtitlePipelineMedia, subtitleCapabilityGate)
	// Activity hub aggregate (UX Redesign D4-1 / ux3-2-1) — composes live scan +
	// batch-subtitle + generation-batch + solo-transcription progress, pending-parse
	// count, download counts, and recent parse events. Wired after the processors
	// since it reads them. Fail-soft per section (B1/F3). transcriptionService
	// (disc-2026-07-transcription-active-jobs) surfaces ad-hoc single-episode/movie
	// jobs that were previously invisible once their progress modal was closed.
	activityService := services.NewActivityService(scannerService, batchProcessor, generationBatchProcessor, transcriptionService, downloadService, repos.ParseJobs)
	activityHandler := handlers.NewActivityHandler(activityService)

	// Home v3 readout-band aggregate (ux3-1-6, tech-spec D1). The four in-flight
	// sources are the SAME instances /activity reads (D2: one counting path with
	// the nav badge); generationBatchProcessor doubles as the live-spend source
	// for the attention cell (D3 precedence: live batch over last persisted run).
	homeSummaryService := services.NewHomeSummaryService(
		repos.Movies, repos.Series, repos.ParseJobs, repos.SubtitleRuns,
		scannerService, batchProcessor, generationBatchProcessor, transcriptionService,
		generationBatchProcessor,
	)
	homeSummaryHandler := handlers.NewHomeSummaryHandler(homeSummaryService)
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
	// Retry-After is NOT one of the seven CORS-safelisted response headers, so a
	// cross-origin frontend (VITE_API_BASE_URL pointed at another host) reads null
	// unless it is exposed explicitly. The login screen counts a lockout down from
	// this header; without it the button stays enabled and every submit 429s with
	// no explanation — the one readout this endpoint exists to hand over.
	corsConfig.ExposeHeaders = []string{"Retry-After"}
	router.Use(cors.New(corsConfig))
	slog.Info("CORS configured", "origins", cfg.CORSOrigins)

	// Register routes
	router.GET("/health", handlers.HealthCheckHandler(db))

	// Auth gate (V0.1.1). A single shared password (VIDO_AUTH_PASSWORD) puts a
	// session in front of the whole API; unset keeps the API open for a LAN-only
	// install. The cookie-signing secret is resolved once at startup (explicit
	// secret → ENCRYPTION_KEY → a random secret persisted under the data dir).
	sessionSecret, err := crypto.SessionSecret(cfg.SessionSecret, cfg.EncryptionKey, cfg.DataDir)
	if err != nil {
		slog.Error("Failed to derive session secret", "error", err)
		os.Exit(1)
	}
	authenticator := handlers.NewAuthenticator(cfg.AuthPassword, sessionSecret)
	authenticator.SetSecureCookie(cfg.SecureCookie)
	authHandler := handlers.NewAuthHandler(authenticator)
	if authenticator.Enabled() {
		slog.Info("🔒 API authentication enabled (VIDO_AUTH_PASSWORD set)")
	} else {
		slog.Warn("⚠️  API authentication DISABLED — VIDO_AUTH_PASSWORD not set. " +
			"Anyone who can reach this server has full control of it. Set VIDO_AUTH_PASSWORD, " +
			"or keep the server strictly on a trusted LAN / behind a VPN.")
	}

	// Auth endpoints (login/logout/status) need neither the database nor an
	// existing session, so they live on their own /api/v1 group OUTSIDE
	// DatabaseGate — login must keep working during a DB outage so the user can
	// still authenticate and see the honest banner.
	authGroup := router.Group("/api/v1")
	authHandler.RegisterRoutes(authGroup)

	// API v1 routes with handler → service → repository architecture.
	// DatabaseGate (bugfix-i-3): while the supervisor reports the database
	// down, every /api/v1 route fails fast with ONE uniform 503
	// DATABASE_UNAVAILABLE instead of ten scattered per-handler failures. The
	// root /health endpoint above stays outside the gate so probes and the
	// frontend banner keep getting the honest detail.
	// AuthGate (V0.1.1): when a password is set, every /api/v1 route requires a
	// valid session cookie (the /api/v1/auth/* endpoints registered above are on
	// their own gate-free group).
	apiV1 := router.Group("/api/v1", handlers.DatabaseGate(dbSupervisor.Healthy), handlers.AuthGate(authenticator))
	{
		movieHandler.RegisterRoutes(apiV1)
		seriesHandler.RegisterRoutes(apiV1)
		doubanRatingHandler.RegisterRoutes(apiV1)  // /{movies,series}/:id/douban-rating (12-1) + /douban-review-summary (12-6)
		logHandler.RegisterRoutes(apiV1)           // Must be before settingsHandler to avoid /settings/:key conflict
		cacheHandler.RegisterRoutes(apiV1)         // Must be before settingsHandler to avoid /settings/:key conflict
		statusHandler.RegisterRoutes(apiV1)        // Must be before settingsHandler to avoid /settings/:key conflict
		statusSummaryHandler.RegisterRoutes(apiV1) // GET /api/v1/status/summary — ambient NAS status strip (ux3-0-3, D4-2)
		activityHandler.RegisterRoutes(apiV1)      // GET /api/v1/activity — Activity hub aggregate (ux3-2-1, D4-1)
		homeSummaryHandler.RegisterRoutes(apiV1)   // GET /api/v1/home-summary — Home v3 readout band (ux3-1-6)
		backupHandler.RegisterRoutes(apiV1)        // Must be before settingsHandler to avoid /settings/:key conflict
		exportHandler.RegisterRoutes(apiV1)        // Must be before settingsHandler to avoid /settings/:key conflict
		settingsHandler.RegisterRoutes(apiV1)
		setupHandler.RegisterRoutes(apiV1)
		mediaHandler.RegisterRoutes(apiV1)
		availabilityHandler.RegisterRoutes(apiV1) // /api/v1/media/check-owned (Story 10-4)
		tmdbHandler.RegisterRoutes(apiV1)
		searchHandler.RegisterRoutes(apiV1) // /api/v1/search — unified instant search (Story 11-3)
		parserHandler.RegisterRoutes(apiV1)
		metadataHandler.RegisterRoutes(apiV1)
		learningHandler.RegisterRoutes(apiV1)
		parseProgressHandler.RegisterRoutes(apiV1)
		handlers.RegisterRetryRoutes(apiV1, retryHandler)
		qbittorrentHandler.RegisterRoutes(apiV1)
		downloadHandler.RegisterRoutes(apiV1)
		libraryHandler.RegisterRoutes(apiV1)
		mediaLibrariesHandler.RegisterRoutes(apiV1) // /api/v1/libraries CRUD (Story 7b-2)
		exploreBlocksHandler.RegisterRoutes(apiV1)  // /api/v1/explore-blocks CRUD + content (Story 10.3)
		filterPresetsHandler.RegisterRoutes(apiV1)  // /api/v1/filter-presets CRUD (Story 11.4)
		requestHandler.RegisterRoutes(apiV1)        // /api/v1/requests create+list (Story 13-1a, Epic 13)
		glossaryHandler.RegisterRoutes(apiV1)       // /api/v1/media/:id/glossary CRUD (Story 9R-15)
		dvrSettingsHandler.RegisterRoutes(apiV1)    // /api/v1/settings/radarr triad + profiles/root-folders passthrough (Story 13-4a)
		recentMediaHandler.RegisterRoutes(apiV1)
		scannerHandler.RegisterRoutes(apiV1)
		subtitleHandler.RegisterRoutes(apiV1)
		generationBatchHandler.RegisterRoutes(apiV1)      // /api/v1/subtitles/generation-batch group (Story 9R-16)
		generationCandidatesHandler.RegisterRoutes(apiV1) // /api/v1/subtitles/generation-candidates (story sub-4-1)
		subtitlePipelineHandler.RegisterRoutes(apiV1)     // POST /api/v1/subtitles/pipeline/run (Story sub-1-6, FR12)
		keySettingsHandler.RegisterRoutes(apiV1)          // GET/PUT /api/v1/settings/keys + POST /test (Story sub-2-1a, FR25)
		transcriptionHandler.RegisterRoutes(apiV1)
		if nfoLocalizer != nil {
			nfoLocalizerHandler.RegisterRoutes(apiV1) // POST /{movies,series,episodes}/:id/localize-nfo (9R-13 + 9R-13a)
		}
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

	// Start database supervisor (bugfix-i-1 / bugfix-i-3)
	dbSupervisorCtx, dbSupervisorCancel := context.WithCancel(context.Background())
	go dbSupervisor.Start(dbSupervisorCtx)

	// Start backup scheduler (Story 6.8)
	schedulerCtx, schedulerCancel := context.WithCancel(context.Background())
	go backupScheduler.Start(schedulerCtx)
	slog.Info("Backup scheduler started")

	// Start scan scheduler (Story 7.2)
	scanSchedulerCtx, scanSchedulerCancel := context.WithCancel(context.Background())
	go scanScheduler.Start(scanSchedulerCtx)
	slog.Info("Scan scheduler started")

	// Start cache sweep scheduler (infra-cache-entries-expiry-sweep)
	cacheSweepCtx, cacheSweepCancel := context.WithCancel(context.Background())
	go cacheSweepScheduler.Start(cacheSweepCtx)
	slog.Info("Cache sweep scheduler started")

	// Start download progress broadcaster (ux3-4-2b — Epic 14 H-1 SSE fan-out)
	downloadProgressCtx, downloadProgressCancel := context.WithCancel(context.Background())
	go downloadProgressBroadcaster.Start(downloadProgressCtx)
	slog.Info("Download progress broadcaster started")

	// Start qBittorrent health monitoring with 30s interval (Story 4.6 - NFR-R6)
	monitorCtx, monitorCancel := context.WithCancel(context.Background())
	go healthMonitor.StartQBMonitoring(monitorCtx)
	slog.Info("qBittorrent health monitoring started (30s interval)")

	// Start the general service health monitor (Story 3.12). This was never
	// wired, so /health/services reported every never-checked service as its
	// factory-default "healthy" — a TMDb with no API key still showed 正常
	// (testsprite-round1 TC088). 5min keeps external pings well under rate
	// limits; the initial sweep runs immediately at startup.
	go healthMonitor.StartMonitoring(monitorCtx, 5*time.Minute)
	slog.Info("Service health monitoring started (5m interval)")

	// Start DVR plugin health scheduler (Story 13-4a — immediate sweep + 60s interval, §7)
	pluginManagerCtx, pluginManagerCancel := context.WithCancel(context.Background())
	if err := pluginManager.Start(pluginManagerCtx); err != nil {
		slog.Error("Failed to start plugin health scheduler", "error", err)
		// Non-fatal — settings endpoints still work; health refreshes on save
	}

	// Start request status poller (Story 13-3a — 15s reconcile loop)
	requestPollerCtx, requestPollerCancel := context.WithCancel(context.Background())
	go requestStatusPoller.Start(requestPollerCtx)

	// Start the subtitle generation worker pool (sub-1-6 AC #3 — fixed
	// concurrency 2 per AD #5/NFR-P3). nil in legacy mode.
	subtitlePipelineCtx, subtitlePipelineCancel := context.WithCancel(context.Background())
	if subtitlePipelinePool != nil {
		if err := subtitlePipelinePool.Start(subtitlePipelineCtx); err != nil {
			slog.Error("Failed to start subtitle pipeline worker pool", "error", err)
			// Non-fatal: the search path and every other feature still work.
		}
	}

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

	// Stop database supervisor (bugfix-i-1)
	slog.Info("Stopping database supervisor...")
	dbSupervisorCancel()

	// Stop scan scheduler (Story 7.2)
	slog.Info("Stopping scan scheduler...")
	scanSchedulerCancel()
	scanScheduler.Stop()

	// Stop cache sweep scheduler (infra-cache-entries-expiry-sweep)
	slog.Info("Stopping cache sweep scheduler...")
	cacheSweepCancel()
	cacheSweepScheduler.Stop()

	// Stop download progress broadcaster (ux3-4-2b)
	slog.Info("Stopping download progress broadcaster...")
	downloadProgressCancel()
	downloadProgressBroadcaster.Stop()

	// Stop request status poller (Story 13-3a)
	slog.Info("Stopping request status poller...")
	requestPollerCancel()
	requestStatusPoller.Stop()

	// Stop subtitle generation worker pool (sub-1-6 AC #3)
	subtitlePipelineCancel()
	if subtitlePipelinePool != nil {
		slog.Info("Stopping subtitle pipeline worker pool...")
		subtitlePipelinePool.Stop()
	}

	// Stop the free-lane auto-generation round (bugfix-autogenerator-no-timeout-
	// or-shutdown AC #6). MUST precede db.Close() below: Stop cancels the
	// in-flight item and waits for it, and that item's failItem cleanup is what
	// keeps the media row out of a stranded `extracting` status.
	if autoGenerator != nil {
		slog.Info("Stopping subtitle auto-generation...")
		autoGenerator.Stop()
	}

	// Stop DVR plugin health scheduler (Story 13-4a)
	slog.Info("Stopping plugin health scheduler...")
	pluginManagerCancel()
	pluginManager.Stop()

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

// extractSlotAdapter narrows *subtitle.ExtractGate to the one method the
// services-side audio extractor needs (Rule 19: services ↛ subtitle, so the
// port lives there and the concrete gate is adapted here).
type extractSlotAdapter struct{ gate *subtitle.ExtractGate }

func (a extractSlotAdapter) Acquire(ctx context.Context) (func(), error) {
	release, _, err := a.gate.Acquire(ctx, nil)
	return release, err
}
