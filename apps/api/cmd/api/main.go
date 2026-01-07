package main

import (
	"context"
	"fmt"
	"log"
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

	// Import migrations to register them via init()
	_ "github.com/vido/api/internal/database/migrations"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode based on environment
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database
	log.Printf("Initializing database at %s", cfg.Database.Path)
	db, err := database.Initialize(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Printf("Database initialized successfully with WAL mode: %v", cfg.Database.WALEnabled)

	// Run database migrations
	log.Printf("Running database migrations...")
	migrationRunner, err := migrations.NewRunner(db.Conn())
	if err != nil {
		log.Fatalf("Failed to create migration runner: %v", err)
	}

	// Register all migrations from global registry
	allMigrations := migrations.GetAll()
	if err := migrationRunner.RegisterAll(allMigrations); err != nil {
		log.Fatalf("Failed to register migrations: %v", err)
	}

	// Apply pending migrations
	ctx := context.Background()
	if err := migrationRunner.Up(ctx); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Check migration status
	status, err := migrationRunner.Status(ctx)
	if err != nil {
		log.Fatalf("Failed to get migration status: %v", err)
	}
	appliedCount := 0
	for _, s := range status {
		if s.Applied {
			appliedCount++
		}
	}
	log.Printf("Database migrations completed: %d/%d applied", appliedCount, len(status))

	// Initialize repositories
	movieRepo := repository.NewMovieRepository(db.Conn())
	seriesRepo := repository.NewSeriesRepository(db.Conn())
	settingsRepo := repository.NewSettingsRepository(db.Conn())

	// Log repository initialization (repositories will be injected into handlers later)
	_ = movieRepo
	_ = seriesRepo
	_ = settingsRepo
	log.Printf("Repositories initialized successfully")

	// Create Gin router
	router := gin.Default()

	// Configure CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:4200"} // React app default port
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	router.Use(cors.New(corsConfig))

	// Register routes
	router.GET("/health", handlers.HealthCheckHandler(db))

	// Start server in a goroutine for graceful shutdown
	addr := cfg.GetAddress()
	log.Printf("Starting Vido API server on %s", addr)

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Run server in goroutine
	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Println("Shutting down server...")

	// Give ongoing requests time to finish
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close database connection
	log.Println("Closing database connection...")
	if err := db.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	<-shutdownCtx.Done()
	log.Println("Server stopped gracefully")
}
