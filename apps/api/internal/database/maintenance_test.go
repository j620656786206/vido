package database

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vido/api/internal/config"
)

func newFileDB(t *testing.T) *DB {
	t.Helper()
	cfg := &config.DatabaseConfig{
		Path:            filepath.Join(t.TempDir(), "maintenance_test.db"),
		WALEnabled:      false,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		BusyTimeout:     5 * time.Second,
		CacheSize:       -2000,
	}
	db, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// bloatDB grows the file with disposable rows and deletes them all, leaving the
// pages on the freelist.
func bloatDB(t *testing.T, db *DB) {
	t.Helper()
	if _, err := db.Exec(`CREATE TABLE bloat (id INTEGER PRIMARY KEY, payload TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	payload := strings.Repeat("x", 4096)
	for i := 0; i < 500; i++ {
		if _, err := db.Exec(`INSERT INTO bloat (payload) VALUES (?)`, payload); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	if _, err := db.Exec(`DELETE FROM bloat`); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestReclaimIfBloated_VacuumsWhenThresholdsMet(t *testing.T) {
	db := newFileDB(t)
	bloatDB(t, db)
	ctx := context.Background()

	before, err := db.pragmaInt(ctx, "freelist_count")
	if err != nil {
		t.Fatalf("freelist_count: %v", err)
	}
	if before == 0 {
		t.Fatal("test setup did not produce any free pages")
	}

	ran, err := db.reclaimIfBloated(ctx, 1, 0.01)
	if err != nil {
		t.Fatalf("reclaimIfBloated: %v", err)
	}
	if !ran {
		t.Fatal("expected VACUUM to run with permissive thresholds")
	}

	after, err := db.pragmaInt(ctx, "freelist_count")
	if err != nil {
		t.Fatalf("freelist_count after vacuum: %v", err)
	}
	if after >= before {
		t.Fatalf("expected freelist to shrink, before=%d after=%d", before, after)
	}
}

func TestReclaimIfBloated_SkipsWhenBelowThresholds(t *testing.T) {
	db := newFileDB(t)
	bloatDB(t, db)
	ctx := context.Background()

	// Absolute-bytes threshold far above what bloatDB frees → skip.
	ran, err := db.reclaimIfBloated(ctx, 1<<40, 0.01)
	if err != nil {
		t.Fatalf("reclaimIfBloated: %v", err)
	}
	if ran {
		t.Fatal("expected VACUUM to be skipped when free bytes are below the floor")
	}

	// Ratio threshold above the actual dead-page share → skip.
	ran, err = db.reclaimIfBloated(ctx, 1, 1.1)
	if err != nil {
		t.Fatalf("reclaimIfBloated: %v", err)
	}
	if ran {
		t.Fatal("expected VACUUM to be skipped when the free ratio is below the floor")
	}
}

func TestReclaimSpaceIfBloated_HealthyFileNoVacuum(t *testing.T) {
	db := newFileDB(t)

	ran, err := db.ReclaimSpaceIfBloated(context.Background())
	if err != nil {
		t.Fatalf("ReclaimSpaceIfBloated: %v", err)
	}
	if ran {
		t.Fatal("a fresh database must not trigger a VACUUM at production thresholds")
	}
}
