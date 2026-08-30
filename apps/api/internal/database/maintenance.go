package database

import (
	"context"
	"fmt"
	"log/slog"
)

// Space-reclaim thresholds (bugfix-system-logs-no-retention). VACUUM rewrites
// the whole file and takes the writer lock, so it only pays off when a large
// share of the file is dead pages — the state the unbounded system_logs table
// left behind (a 245MB file holding 1MB of live data). Both conditions must
// hold before we VACUUM.
const (
	// reclaimMinFreeBytes is the minimum absolute freelist size worth a
	// rewrite. Below this the file is small enough that the bloat is harmless.
	reclaimMinFreeBytes = int64(32 << 20) // 32MB
	// reclaimMinFreeRatio is the minimum share of the file that must be dead
	// pages. Below this the file is mostly live data and VACUUM would churn
	// disk for little gain.
	reclaimMinFreeRatio = 0.25
)

// ReclaimSpaceIfBloated runs VACUUM when the freelist says the database file is
// mostly dead pages. It is intended for STARTUP ONLY — after migrations, before
// the HTTP listener and schedulers start — because VACUUM blocks writers for
// the duration of the rewrite; that window is the one time nothing contends.
// Returns whether a VACUUM ran.
func (db *DB) ReclaimSpaceIfBloated(ctx context.Context) (bool, error) {
	return db.reclaimIfBloated(ctx, reclaimMinFreeBytes, reclaimMinFreeRatio)
}

// reclaimIfBloated is the threshold-injectable core of ReclaimSpaceIfBloated.
func (db *DB) reclaimIfBloated(ctx context.Context, minFreeBytes int64, minFreeRatio float64) (bool, error) {
	pageCount, err := db.pragmaInt(ctx, "page_count")
	if err != nil {
		return false, err
	}
	freeCount, err := db.pragmaInt(ctx, "freelist_count")
	if err != nil {
		return false, err
	}
	pageSize, err := db.pragmaInt(ctx, "page_size")
	if err != nil {
		return false, err
	}

	if pageCount == 0 {
		return false, nil
	}
	freeBytes := freeCount * pageSize
	freeRatio := float64(freeCount) / float64(pageCount)
	if freeBytes < minFreeBytes || freeRatio < minFreeRatio {
		return false, nil
	}

	slog.Info("Database file is mostly dead pages — reclaiming space with VACUUM",
		"file_bytes", pageCount*pageSize,
		"free_bytes", freeBytes,
		"free_ratio", fmt.Sprintf("%.2f", freeRatio),
	)
	if _, err := db.conn.ExecContext(ctx, "VACUUM"); err != nil {
		return false, fmt.Errorf("vacuum: %w", err)
	}

	afterPages, err := db.pragmaInt(ctx, "page_count")
	if err != nil {
		// The VACUUM itself succeeded; the follow-up size read is best-effort.
		slog.Warn("VACUUM done but post-vacuum page_count read failed", "error", err)
		return true, nil
	}
	slog.Info("Database space reclaimed",
		"before_bytes", pageCount*pageSize,
		"after_bytes", afterPages*pageSize,
	)
	return true, nil
}

// pragmaInt reads a single integer-valued PRAGMA.
func (db *DB) pragmaInt(ctx context.Context, name string) (int64, error) {
	var v int64
	if err := db.conn.QueryRowContext(ctx, "PRAGMA "+name).Scan(&v); err != nil {
		return 0, fmt.Errorf("pragma %s: %w", name, err)
	}
	return v, nil
}
