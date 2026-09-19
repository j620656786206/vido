package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/segkey"
)

// ─── disc-2026-09-generation-resume-a-translation-cache AC #2 / #5 ─────────

func newMigratedServicesTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

func TestSegmentStore_TagsTheFamilyAndMapsValues(t *testing.T) {
	repo := &fakeCacheRepo{}
	store := NewSegmentStore(repo)
	ctx := context.Background()

	require.NoError(t, store.Set(ctx, "subseg:v1:present", "早安", segkey.TTL))
	assert.Equal(t, segkey.Type, repo.lastType,
		"a wrong tag would split this leg's entries from the extract leg's and orphan them from ClearByType")
	assert.Equal(t, segkey.TTL, repo.lastTTL)

	values, err := store.GetMany(ctx, []string{"subseg:v1:present", "subseg:v1:absent"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"subseg:v1:present": "早安"}, values,
		"a miss is expressed by absence, and a hit returns the stored value verbatim")
}

// Both legs must land in the SAME table rows: a cue cached by the extract leg
// is readable by this one and vice versa. Anything else means two caches.
func TestSegmentStore_SharesRowsWithTheExtractLeg(t *testing.T) {
	db := newMigratedServicesTestDB(t)
	cache := repository.NewCacheRepository(db)
	ctx := context.Background()

	require.NoError(t, NewSegmentStore(cache).Set(ctx, "subseg:v1:shared", "早安", segkey.TTL))

	var cacheType string
	require.NoError(t, db.QueryRow(`SELECT type FROM cache_entries WHERE key = ?`, "subseg:v1:shared").Scan(&cacheType))
	assert.Equal(t, "subtitle_segment", cacheType)

	values, err := NewSegmentStore(cache).GetMany(ctx, []string{"subseg:v1:shared"})
	require.NoError(t, err)
	assert.Equal(t, "早安", values["subseg:v1:shared"])
}

// AC #5 scale gate: 1,000 half-finished films leave ~200k rows behind. The
// per-run read must stay a bounded primary-key lookup, not a table scan.
func TestSegmentStore_ReadStaysFastAgainst200kRows(t *testing.T) {
	if testing.Short() {
		t.Skip("scale gate — skipped in -short")
	}
	db := newMigratedServicesTestDB(t)
	ctx := context.Background()

	// Bulk insert directly: 200k round trips through the repository would make
	// the FIXTURE the slow part, not the thing under test.
	tx, err := db.Begin()
	require.NoError(t, err)
	stmt, err := tx.Prepare(`INSERT INTO cache_entries (key, value, type, expires_at) VALUES (?, ?, ?, ?)`)
	require.NoError(t, err)
	expires := time.Now().Add(segkey.TTL)
	for i := 0; i < 200_000; i++ {
		typ := segkey.Type
		if i%5 == 0 { // a mixed table, as in production
			typ = "metadata"
		}
		_, err := stmt.Exec(fmt.Sprintf("subseg:v1:row-%06d", i), "譯文", typ, expires)
		require.NoError(t, err)
	}
	require.NoError(t, stmt.Close())
	require.NoError(t, tx.Commit())

	keys := make([]string, 2000)
	for i := range keys {
		keys[i] = fmt.Sprintf("subseg:v1:row-%06d", i*7)
	}

	start := time.Now()
	values, err := NewSegmentStore(repository.NewCacheRepository(db)).GetMany(ctx, keys)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, values, 2000)
	assert.Lessf(t, elapsed, 100*time.Millisecond,
		"a 2,000-cue film's cache read took %v against 200k rows — the resume feature would cost more than it saves", elapsed)
	t.Logf("GetMany(2000 keys) over 200k rows: %v", elapsed)
}
