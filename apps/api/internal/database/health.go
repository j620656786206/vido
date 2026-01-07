package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// HealthStatus represents the health status of the database
type HealthStatus struct {
	Status          string        `json:"status"`           // "healthy", "degraded", "unhealthy"
	Latency         time.Duration `json:"latency"`          // Time taken for ping
	WALEnabled      bool          `json:"wal_enabled"`      // Whether WAL mode is active
	WALMode         string        `json:"wal_mode"`         // Current journal mode
	SyncMode        string        `json:"sync_mode"`        // Current synchronous mode
	OpenConnections int           `json:"open_connections"` // Current open connections
	IdleConnections int           `json:"idle_connections"` // Current idle connections
	Error           string        `json:"error,omitempty"`  // Error message if unhealthy
}

// Health performs a comprehensive health check on the database
func (db *DB) Health(ctx context.Context) *HealthStatus {
	health := &HealthStatus{
		Status: "healthy",
	}

	// Measure ping latency
	start := time.Now()
	if err := db.pingWithContext(ctx); err != nil {
		health.Status = "unhealthy"
		health.Error = fmt.Sprintf("ping failed: %v", err)
		health.Latency = time.Since(start)
		return health
	}
	health.Latency = time.Since(start)

	// Get connection stats
	stats := db.Stats()
	health.OpenConnections = stats.OpenConnections
	health.IdleConnections = stats.Idle

	// Check WAL mode
	walMode, err := db.GetWALMode()
	if err != nil {
		health.Status = "degraded"
		health.Error = fmt.Sprintf("failed to get WAL mode: %v", err)
	} else {
		health.WALMode = walMode
		health.WALEnabled = (walMode == "wal")
	}

	// Check sync mode
	syncMode, err := db.GetSyncMode()
	if err != nil {
		if health.Status == "healthy" {
			health.Status = "degraded"
		}
		health.Error = fmt.Sprintf("%s; failed to get sync mode: %v", health.Error, err)
	} else {
		health.SyncMode = syncMode
	}

	// If WAL is expected but not enabled, mark as degraded
	if db.config.WALEnabled && !health.WALEnabled {
		health.Status = "degraded"
		if health.Error == "" {
			health.Error = "WAL mode is configured but not active"
		}
	}

	return health
}

// HealthCheck performs a quick health check (ping only)
func (db *DB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.pingWithContext(ctx)
}

// IsHealthy returns true if the database is healthy
func (db *DB) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health := db.Health(ctx)
	return health.Status == "healthy"
}

// pingWithContext performs a ping operation with the given context
func (db *DB) pingWithContext(ctx context.Context) error {
	if db.conn == nil {
		return fmt.Errorf("database connection is nil")
	}

	if err := db.conn.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// GetConnectionStats returns detailed connection pool statistics
func (db *DB) GetConnectionStats() sql.DBStats {
	return db.Stats()
}

// QuickHealth returns a simple health status suitable for endpoints
func (db *DB) QuickHealth() map[string]interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	health := db.Health(ctx)

	result := map[string]interface{}{
		"status":  health.Status,
		"latency": health.Latency.Milliseconds(),
	}

	if health.Error != "" {
		result["error"] = health.Error
	}

	return result
}
