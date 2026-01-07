package middleware

import (
	"os"
	"time"

	"github.com/alexyu/vido/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	// RequestIDHeader is the header name for request ID
	RequestIDHeader = "X-Request-ID"
)

// InitLogger configures the global logger based on config
func InitLogger(cfg *config.Config) {
	// Configure zerolog output format
	if cfg.IsDevelopment() {
		// Pretty console output for development
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	} else {
		// JSON output for production
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	// Set log level
	switch cfg.LogLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// Logger returns a middleware that logs HTTP requests with structured JSON logging
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate or retrieve request ID
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header(RequestIDHeader, requestID)

		// Record start time
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		// Construct full path with query string
		if raw != "" {
			path = path + "?" + raw
		}

		// Log the request
		logEvent := log.Info()
		if statusCode >= 500 {
			logEvent = log.Error()
		} else if statusCode >= 400 {
			logEvent = log.Warn()
		}

		logEvent.
			Str("request_id", requestID).
			Str("method", method).
			Str("path", path).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("client_ip", clientIP).
			Int("body_size", c.Writer.Size()).
			Str("user_agent", c.Request.UserAgent()).
			Msg("HTTP request")
	}
}

// RequestID returns a middleware that ensures every request has a request ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate or retrieve request ID
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}
