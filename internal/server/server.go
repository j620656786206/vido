package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/alexyu/vido/internal/config"
	"github.com/alexyu/vido/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Server represents the HTTP server
type Server struct {
	config *config.Config
	router *gin.Engine
	server *http.Server
}

// New creates a new Server instance
func New(cfg *config.Config) *Server {
	// Initialize logger
	middleware.InitLogger(cfg)

	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Create router without default middleware (we'll add our own)
	router := gin.New()

	// Apply global middleware in order
	router.Use(middleware.Recovery(cfg)) // Panic recovery (must be first)
	router.Use(middleware.Logger())      // Structured logging
	router.Use(middleware.CORS(cfg))     // CORS headers
	router.Use(middleware.ErrorHandler(cfg)) // Error handling

	// Create HTTP server
	httpServer := &http.Server{
		Addr:           cfg.GetAddress(),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	s := &Server{
		config: cfg,
		router: router,
		server: httpServer,
	}

	// Setup routes
	s.setupRoutes()

	return s
}

// Start begins listening and serving HTTP requests
func (s *Server) Start() error {
	log.Info().
		Str("address", s.config.GetAddress()).
		Str("environment", s.config.Env).
		Msg("Starting server")

	// Start server in a goroutine so it doesn't block
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("Shutting down server gracefully...")

	// Attempt graceful shutdown
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Info().Msg("Server stopped")
	return nil
}

// Router returns the Gin router for configuration
func (s *Server) Router() *gin.Engine {
	return s.router
}
