package services

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// bugfix-restore-fails-on-real-database — restore on a REAL schema.

// seedLibrary puts rows with foreign keys in place: a library, its path, and a
// movie that belongs to it. Clearing media_libraries first is what broke the
// old table-by-table restore.
func seedLibrary(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, q := range []string{
		`INSERT INTO media_libraries (id, name, content_type) VALUES ('lib1', '電影', 'movie')`,
		`INSERT INTO media_library_paths (id, library_id, path) VALUES ('p1', 'lib1', '/media/movies')`,
		`INSERT INTO movies (id, title, release_date, library_id) VALUES ('m1', '你的名字', '2016-08-26', 'lib1')`,
		`INSERT INTO movies (id, title, release_date, library_id) VALUES ('m2', '天氣之子', '2019-07-19', 'lib1')`,
	} {
		_, err := db.Exec(q)
		require.NoError(t, err, q)
	}
}

func titles(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT title FROM movies ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		out = append(out, s)
	}
	return out
}

// backupThenRestoreFixture backs up a seeded real database and wires the repo
// mock so RestoreBackup can find that backup (and create its snapshot).
func backupThenRestoreFixture(t *testing.T) (*BackupService, *sql.DB, *models.Backup) {
	t.Helper()
	db := openVidoTestDB(t, filepath.Join(t.TempDir(), "vido.db"), 0)
	t.Cleanup(func() { db.Close() })
	seedLibrary(t, db)

	repo := backupRepoMock()
	svc := NewBackupService(db, repo, t.TempDir())
	b, err := svc.CreateBackup(context.Background())
	require.NoError(t, err)
	// The pre-restore snapshot is created (then updated) during the restore;
	// mirror its latest state so the rollback can look it up by ID.
	snap := &models.Backup{}
	keep := mock.MatchedBy(func(x *models.Backup) bool {
		if x.ID != b.ID {
			*snap = *x
		}
		return true
	})
	repo.ExpectedCalls = nil
	repo.On("Create", mock.Anything, keep).Return(nil)
	repo.On("Update", mock.Anything, keep).Return(nil)
	repo.On("GetByID", mock.Anything, b.ID).Return(b, nil)
	repo.On("GetByID", mock.Anything, mock.MatchedBy(func(id string) bool { return id != b.ID })).
		Return(snap, nil).Maybe()
	return svc, db, b
}

func TestRestore_RealSchemaWithForeignKeys(t *testing.T) {
	svc, db, b := backupThenRestoreFixture(t)

	// Change things after the backup.
	_, err := db.Exec(`DELETE FROM movies WHERE id = 'm2'`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE movies SET title = '改過的' WHERE id = 'm1'`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO movies (id, title, release_date, library_id) VALUES ('m3', '之後才加', '2020-01-01', 'lib1')`)
	require.NoError(t, err)

	res, err := svc.RestoreBackup(context.Background(), b.ID)
	require.NoError(t, err)
	require.Equal(t, models.RestoreStatusCompleted, res.Status, res.Error)
	assert.Equal(t, []string{"你的名字", "天氣之子"}, titles(t, db))

	var fkProblems int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&fkProblems))
	assert.Zero(t, fkProblems)
}

func TestRestore_TwiceInARowNeverTripsOverItself(t *testing.T) {
	svc, db, b := backupThenRestoreFixture(t)
	for i := 0; i < 2; i++ {
		res, err := svc.RestoreBackup(context.Background(), b.ID)
		require.NoError(t, err)
		require.Equal(t, models.RestoreStatusCompleted, res.Status, "round %d: %s", i, res.Error)
	}
	assert.Equal(t, []string{"你的名字", "天氣之子"}, titles(t, db))
}

func TestRestore_OlderSchemaIsBroughtUpToDate(t *testing.T) {
	db, _ := createTestDB(t)
	defer db.Close()
	backupDir := t.TempDir()
	filename, checksum := createTestBackupArchive(t, backupDir, 20, "from-v20")
	repo := backupRepoMock()
	b := &models.Backup{ID: "old", Filename: filename, Checksum: checksum, Status: models.BackupStatusCompleted}
	repo.On("GetByID", mock.Anything, "old").Return(b, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Return((*models.Backup)(nil), nil).Maybe()
	svc := NewBackupService(db, repo, backupDir)

	res, err := svc.RestoreBackup(context.Background(), "old")
	require.NoError(t, err)
	require.Equal(t, models.RestoreStatusCompleted, res.Status, res.Error)

	var got string
	require.NoError(t, db.QueryRow(`SELECT name FROM test_data WHERE id = 'restored'`).Scan(&got))
	assert.Equal(t, "from-v20", got)
	var v int64
	require.NoError(t, db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&v))
	assert.Equal(t, latestMigrationVersion(), v, "an older backup is migrated up after the swap")
}

func TestRestore_NewerBackupIsRefusedAndNothingChanges(t *testing.T) {
	db, _ := createTestDB(t)
	defer db.Close()
	backupDir := t.TempDir()
	// The manifest would say 17 (old archives lied); the DB inside says newer.
	filename, checksum := createTestBackupArchive(t, backupDir, latestMigrationVersion()+5, "future")
	repo := backupRepoMock()
	b := &models.Backup{ID: "new", Filename: filename, Checksum: checksum, Status: models.BackupStatusCompleted}
	repo.On("GetByID", mock.Anything, "new").Return(b, nil)
	repo.On("GetByID", mock.Anything, mock.Anything).Return((*models.Backup)(nil), nil).Maybe()
	svc := NewBackupService(db, repo, backupDir)

	res, err := svc.RestoreBackup(context.Background(), "new")
	require.NoError(t, err)
	assert.Equal(t, models.RestoreStatusFailed, res.Status)
	assert.Contains(t, res.Error, "RESTORE_INCOMPATIBLE_VERSION")
	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM test_data WHERE id = 'restored'`).Scan(&n))
	assert.Zero(t, n, "the current database is untouched")
}

func TestRestore_FailedMigrationRollsBackToTheSnapshot(t *testing.T) {
	svc, db, b := backupThenRestoreFixture(t)
	_, err := db.Exec(`UPDATE movies SET title = '還原前的樣子' WHERE id = 'm1'`)
	require.NoError(t, err)

	calls := 0
	svc.migrateUp = func(ctx context.Context, d *sql.DB) error {
		calls++
		if calls == 1 {
			return errors.New("boom")
		}
		return runAllMigrations(ctx, d)
	}

	res, err := svc.RestoreBackup(context.Background(), b.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RestoreStatusFailed, res.Status)
	assert.Contains(t, res.Error, "RESTORE_ROLLBACK_SUCCESS")
	assert.NotContains(t, res.Error, "already attached")
	assert.Contains(t, titles(t, db), "還原前的樣子", "rolled back to the pre-restore snapshot")
}

func TestBackup_UsesTheRealSchemaVersion(t *testing.T) {
	db := openVidoTestDB(t, filepath.Join(t.TempDir(), "vido.db"), 0)
	defer db.Close()
	backupDir := t.TempDir()
	svc := NewBackupService(db, backupRepoMock(), backupDir)
	b, err := svc.CreateBackup(context.Background())
	require.NoError(t, err)
	assert.Equal(t, latestMigrationVersion(), b.SchemaVersion)
	_, m, _ := archiveEntries(t, filepath.Join(backupDir, b.Filename))
	assert.Equal(t, latestMigrationVersion(), m.SchemaVersion)
}

// Another part of the app is mid-write when the user presses 還原: the restore
// must wait for it, not fail (/ship CR).
func TestRestore_WaitsForAConcurrentWriter(t *testing.T) {
	svc, db, b := backupThenRestoreFixture(t)

	tx, err := db.Begin()
	require.NoError(t, err)
	_, err = tx.Exec(`UPDATE movies SET title = '寫到一半' WHERE id = 'm1'`)
	require.NoError(t, err)
	released := make(chan struct{})
	go func() {
		time.Sleep(700 * time.Millisecond)
		_ = tx.Commit()
		close(released)
	}()

	res, err := svc.RestoreBackup(context.Background(), b.ID)
	<-released
	require.NoError(t, err)
	require.Equal(t, models.RestoreStatusCompleted, res.Status, res.Error)
	assert.Equal(t, []string{"你的名字", "天氣之子"}, titles(t, db))
}
