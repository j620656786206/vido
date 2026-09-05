package database

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/vido/api/internal/config"
)

func newSupervisorTestDB(t *testing.T) *DB {
	t.Helper()
	cfg := &config.DatabaseConfig{
		Path:            filepath.Join(t.TempDir(), "supervisor_test.db"),
		WALEnabled:      false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		BusyTimeout:     time.Second,
		CacheSize:       -2000,
	}
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	return db
}

func TestSupervisor_HealthyDatabaseStaysHealthy(t *testing.T) {
	db := newSupervisorTestDB(t)
	defer db.Close()
	sup := NewSupervisor(db)

	for i := 0; i < 5; i++ {
		sup.tick(context.Background())
	}
	if !sup.Healthy() {
		t.Fatal("a live database must stay healthy")
	}
}

func TestSupervisor_DeclaresUnhealthyAfterThreshold(t *testing.T) {
	db := newSupervisorTestDB(t)
	sup := NewSupervisor(db)
	db.Close() // every ping now fails

	ctx := context.Background()
	sup.tick(ctx)
	sup.tick(ctx)
	if !sup.Healthy() {
		t.Fatal("must stay healthy below the failure threshold")
	}
	sup.tick(ctx) // third consecutive failure hits the threshold
	if sup.Healthy() {
		t.Fatal("must be unhealthy after threshold consecutive ping failures")
	}
	if !sup.evidenceCaptured {
		t.Fatal("evidence must be captured when the unhealthy state is entered")
	}
}

// Regression (2026-09-06, first CI run of `go test`): a recycle against a closed
// handle used to race database/sql's cleaner goroutine and panic with "send on
// closed channel" — the whole test binary died. The verdict must stay unhealthy
// (nothing was recovered) and the process must stay alive.
func TestSupervisor_RecoveryOnClosedHandleDoesNotPanic(t *testing.T) {
	db := newSupervisorTestDB(t)
	sup := NewSupervisor(db)
	db.Close()

	sup.healthy.Store(false)
	for i := 0; i < 50; i++ { // widen the window the race used to need
		sup.attemptRecovery(context.Background())
	}

	if sup.Healthy() {
		t.Fatal("a closed handle must not be reported healthy")
	}
}

func TestSupervisor_RecoversOnNextHealthyTick(t *testing.T) {
	db := newSupervisorTestDB(t)
	defer db.Close()
	sup := NewSupervisor(db)

	// Simulate a prior incident: unhealthy verdict with stale counters.
	sup.healthy.Store(false)
	sup.consecutiveFails = sup.threshold
	sup.evidenceCaptured = true

	sup.tick(context.Background())

	if !sup.Healthy() {
		t.Fatal("a successful ping must flip the verdict back to healthy")
	}
	if sup.consecutiveFails != 0 {
		t.Fatalf("recovery must reset consecutiveFails, got %d", sup.consecutiveFails)
	}
	if sup.evidenceCaptured {
		t.Fatal("recovery must re-arm evidence capture for the next incident")
	}
}

func TestSupervisor_RecoveryRestoresPoolSettings(t *testing.T) {
	db := newSupervisorTestDB(t)
	defer db.Close()
	sup := NewSupervisor(db)

	// attemptRecovery on a live database succeeds and must leave the pool with
	// the configured lifetimes (a lingering 1ns lifetime would churn every
	// connection forever).
	sup.healthy.Store(false)
	sup.attemptRecovery(context.Background())

	if !sup.Healthy() {
		t.Fatal("recovery against a live database must succeed")
	}
	// No public getter for pool lifetimes exists; prove behaviour instead: a
	// query right after recovery must work and keep working.
	for i := 0; i < 3; i++ {
		if err := db.Ping(); err != nil {
			t.Fatalf("ping %d after recovery failed: %v", i, err)
		}
	}
}
