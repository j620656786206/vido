package database

import (
	"context"
	"database/sql"
	"log/slog"
	"sync/atomic"
	"time"
)

// Supervisor watches database liveness in the background and owns the recovery
// path (bugfix-i-1-sqlite-permanent-unhealthy / bugfix-i-3-db-dead-returns-200).
//
// The NAS incident it exists for: SQLite on an unreliable mount entered a
// PERMANENTLY unhealthy state (ping deadline-exceeded, open_connections 0) that
// only a reinstall cleared — and the evidence was destroyed by that reinstall.
// The supervisor turns that into three guarantees:
//
//  1. Honesty: Healthy() is the single source of truth the API gate and the
//     health endpoints read, so a dead database looks like ONE understandable
//     condition instead of ten scattered failures.
//  2. Evidence: on entering the unhealthy state it captures PRAGMA and pool
//     state to the console log (stderr survives a dead database) BEFORE any
//     recovery attempt, so the next incident is diagnosable.
//  3. Recovery: it force-recycles every pooled connection and re-pings each
//     tick. A stale FUSE/shfs lock held by a dead pooled connection heals
//     without a container reinstall once the underlying mount recovers.
type Supervisor struct {
	db        *DB
	interval  time.Duration
	threshold int // consecutive ping failures before declaring unhealthy

	healthy          atomic.Bool
	consecutiveFails int
	evidenceCaptured bool
}

const (
	defaultSupervisorInterval  = 30 * time.Second
	defaultSupervisorThreshold = 3
	supervisorPingTimeout      = 5 * time.Second
)

// NewSupervisor creates a Supervisor for db. The database starts out assumed
// healthy (main.go already pinged it during Initialize).
func NewSupervisor(db *DB) *Supervisor {
	s := &Supervisor{
		db:        db,
		interval:  defaultSupervisorInterval,
		threshold: defaultSupervisorThreshold,
	}
	s.healthy.Store(true)
	return s
}

// Healthy reports the supervisor's current verdict. It is a cached atomic read
// — safe to call on every request (the API gate does).
func (s *Supervisor) Healthy() bool {
	return s.healthy.Load()
}

// Start runs the watch loop until ctx is cancelled. It blocks; callers run it
// in a dedicated goroutine.
func (s *Supervisor) Start(ctx context.Context) {
	slog.Info("Database supervisor started",
		"interval", s.interval, "fail_threshold", s.threshold)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

// tick performs one liveness check and drives the state machine.
func (s *Supervisor) tick(ctx context.Context) {
	if s.ping(ctx) == nil {
		if !s.healthy.Load() {
			slog.Info("Database recovered — resuming normal service")
		}
		s.healthy.Store(true)
		s.consecutiveFails = 0
		s.evidenceCaptured = false
		return
	}

	s.consecutiveFails++
	if s.consecutiveFails < s.threshold {
		slog.Warn("Database ping failed",
			"consecutive_fails", s.consecutiveFails, "fail_threshold", s.threshold)
		return
	}

	// Threshold reached: declare unhealthy, capture evidence ONCE per
	// incident, then attempt recovery every tick until the ping comes back.
	first := s.healthy.Swap(false) // true only on the tick that flips the state
	if first || !s.evidenceCaptured {
		s.captureEvidence(ctx)
		s.evidenceCaptured = true
	}
	s.attemptRecovery(ctx)
}

func (s *Supervisor) ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, supervisorPingTimeout)
	defer cancel()
	return s.db.pingWithContext(pingCtx)
}

// captureEvidence logs everything diagnosable about the incident to the
// console handler BEFORE recovery mutates any state. This is the 保留現場 the
// last incident never got: the reinstall that "fixed" it destroyed the state.
func (s *Supervisor) captureEvidence(ctx context.Context) {
	stats := s.db.Stats()
	attrs := []any{
		"db_path", s.db.config.Path,
		"open_connections", stats.OpenConnections,
		"idle_connections", stats.Idle,
		"in_use", stats.InUse,
		"wait_count", stats.WaitCount,
		"wait_duration", stats.WaitDuration.String(),
		"max_open_connections", stats.MaxOpenConnections,
	}
	// PRAGMA reads are best-effort — on a dead database they time out fast and
	// record that fact, which is itself evidence.
	for _, pragma := range []string{"journal_mode", "busy_timeout", "wal_autocheckpoint"} {
		attrs = append(attrs, "pragma_"+pragma, s.pragmaEvidence(ctx, pragma))
	}
	slog.Error("DATABASE_UNHEALTHY: liveness lost — evidence snapshot before recovery", attrs...)
}

func (s *Supervisor) pragmaEvidence(ctx context.Context, name string) string {
	pragmaCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var v string
	if err := s.db.conn.QueryRowContext(pragmaCtx, "PRAGMA "+name).Scan(&v); err != nil {
		return "unreadable: " + err.Error()
	}
	return v
}

// attemptRecovery force-recycles every pooled connection, then re-pings. The
// incident's signature (open_connections 0, every ping timing out, only a
// reinstall clearing it) matches a pool full of connections wedged on a dead
// mount: database/sql happily keeps handing them back. Expiring the pool makes
// the next ping dial a FRESH connection, which succeeds as soon as the mount
// itself is back.
func (s *Supervisor) attemptRecovery(ctx context.Context) {
	cfg := s.db.config
	conn := s.db.conn

	// A closed handle is shutdown, not an incident: there is no pool left to
	// recycle, and SetConnMaxLifetime on a closed *sql.DB can panic the process
	// (it sends on the cleaner channel that Close just closed). Hold closeMu for
	// the whole recycle so Close cannot slip in between the lifetime pokes.
	s.db.closeMu.Lock()
	defer s.db.closeMu.Unlock()
	if s.db.closed {
		slog.Warn("Database handle already closed — skipping pool recycle")
		return
	}

	// Expire everything: with a 1ns lifetime, every pooled connection is
	// already expired the moment it is next requested, so the pool closes it
	// and dials fresh. The short lifetime MUST stay in force through the
	// verification ping — restoring it first would re-validate the wedged
	// connections before anything discarded them (expiry is checked lazily
	// against the CURRENT setting).
	conn.SetConnMaxLifetime(time.Nanosecond)
	conn.SetConnMaxIdleTime(time.Nanosecond)
	err := s.ping(ctx)
	conn.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	conn.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	if err != nil {
		slog.Warn("Database recovery attempt failed — will retry",
			"error", err, "retry_in", s.interval.String())
		return
	}

	s.healthy.Store(true)
	s.consecutiveFails = 0
	s.evidenceCaptured = false
	slog.Info("Database recovered after connection-pool recycle")
}

// Stats re-export so callers can enrich health payloads without reaching into
// the wrapped connection.
func (s *Supervisor) Stats() sql.DBStats {
	return s.db.Stats()
}
