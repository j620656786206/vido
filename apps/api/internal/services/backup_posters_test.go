package services

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// bugfix-backup-includes-uploaded-posters -------------------------------------

func writePoster(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

// archiveEntries lists an archive's entry names and the manifest it carries.
func archiveEntries(t *testing.T, path string) ([]string, backupManifest, map[string]string) {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	tr := tar.NewReader(gz)
	var names []string
	var m backupManifest
	contents := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, h.Name)
		b, err := io.ReadAll(tr)
		require.NoError(t, err)
		if h.Name == "manifest.json" {
			require.NoError(t, json.Unmarshal(b, &m))
		} else if h.Typeflag == tar.TypeReg {
			contents[h.Name] = string(b)
		}
	}
	sort.Strings(names)
	return names, m, contents
}

func backupRepoMock() *MockBackupRepo {
	repo := new(MockBackupRepo)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*models.Backup")).Return(nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*models.Backup")).Return(nil)
	return repo
}

func TestBackupService_CreateBackup_IncludesOnlyUploadedPosters(t *testing.T) {
	db, _ := createTestDB(t)
	defer db.Close()
	backupDir, posterDir := t.TempDir(), t.TempDir()
	writePoster(t, posterDir, "m1.jpg", "POSTER")
	writePoster(t, posterDir, "m1-thumb.jpg", "THUMB")
	writePoster(t, posterDir, "m1.jpg.bak", "PARKED")
	writePoster(t, posterDir, "notes.txt", "NOPE")
	require.NoError(t, os.Mkdir(filepath.Join(posterDir, "sub.jpg"), 0o755))
	outside := filepath.Join(t.TempDir(), "outside.jpg")
	require.NoError(t, os.WriteFile(outside, []byte("OUT"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(posterDir, "link.jpg")))

	svc := NewBackupService(db, backupRepoMock(), backupDir)
	svc.SetPosterDir(posterDir)
	b, err := svc.CreateBackup(context.Background())
	require.NoError(t, err)

	names, m, contents := archiveEntries(t, filepath.Join(backupDir, b.Filename))
	assert.Equal(t, []string{"manifest.json", "posters/", "posters/m1-thumb.jpg", "posters/m1.jpg", "vido.db"}, names)
	assert.Equal(t, 2, m.Posters)
	assert.Equal(t, "POSTER", contents["posters/m1.jpg"])
}

func TestBackupService_CreateBackup_NoPosterFolderStillBacksUp(t *testing.T) {
	db, _ := createTestDB(t)
	defer db.Close()
	backupDir := t.TempDir()
	svc := NewBackupService(db, backupRepoMock(), backupDir)
	svc.SetPosterDir(filepath.Join(t.TempDir(), "posters")) // never created
	b, err := svc.CreateBackup(context.Background())
	require.NoError(t, err)
	names, m, _ := archiveEntries(t, filepath.Join(backupDir, b.Filename))
	assert.Equal(t, []string{"manifest.json", "vido.db"}, names)
	assert.Equal(t, 0, m.Posters)
}

// restoreFixture backs up with posterDir holding m1 (+thumb), then lets the
// caller change the folder before restoring that backup.
func restoreFixture(t *testing.T) (*BackupService, string, *models.Backup, *MockBackupRepo) {
	t.Helper()
	db, _ := createTestDB(t)
	t.Cleanup(func() { db.Close() })
	backupDir, posterDir := t.TempDir(), t.TempDir()
	writePoster(t, posterDir, "m1.jpg", "ORIGINAL")
	writePoster(t, posterDir, "m1-thumb.jpg", "ORIGINAL-THUMB")
	repo := backupRepoMock()
	svc := NewBackupService(db, repo, backupDir)
	svc.SetPosterDir(posterDir)
	b, err := svc.CreateBackup(context.Background())
	require.NoError(t, err)
	repo.On("GetByID", mock.Anything, b.ID).Return(b, nil)
	return svc, posterDir, b, repo
}

func TestBackupService_Restore_PutsPostersBack(t *testing.T) {
	svc, posterDir, b, _ := restoreFixture(t)
	writePoster(t, posterDir, "m1.jpg", "CHANGED")
	require.NoError(t, os.Remove(filepath.Join(posterDir, "m1-thumb.jpg")))
	writePoster(t, posterDir, "extra.jpg", "NEWER UPLOAD")
	long := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(posterDir, "extra.jpg"), long, long))

	res, err := svc.RestoreBackup(context.Background(), b.ID)
	require.NoError(t, err)
	require.Equal(t, models.RestoreStatusCompleted, res.Status, res.Error)
	assert.Equal(t, 2, res.PostersRestored)
	assert.Equal(t, 0, res.PostersFailed)
	assert.Equal(t, "還原完成，資料庫與 2 張上傳的海報已恢復", res.Message)

	got, _ := os.ReadFile(filepath.Join(posterDir, "m1.jpg"))
	assert.Equal(t, "ORIGINAL", string(got))
	got, _ = os.ReadFile(filepath.Join(posterDir, "m1-thumb.jpg"))
	assert.Equal(t, "ORIGINAL-THUMB", string(got))
	_, err = os.Stat(filepath.Join(posterDir, "extra.jpg"))
	assert.NoError(t, err, "a restore puts posters back; it never empties the folder")

	info, err := os.Stat(filepath.Join(posterDir, "m1.jpg"))
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), info.ModTime(), time.Minute,
		"restored files look fresh so the orphan sweep's grace window covers them")
	assert.ElementsMatch(t, []string{"m1.jpg", "m1-thumb.jpg", "extra.jpg"}, dirNames(t, posterDir),
		"no temp files left behind")
}

func TestBackupService_Restore_OldBackupLeavesPostersAlone(t *testing.T) {
	db, _ := createTestDB(t)
	defer db.Close()
	backupDir, posterDir := t.TempDir(), t.TempDir()
	writePoster(t, posterDir, "m1.jpg", "CURRENT")
	long := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(posterDir, "m1.jpg"), long, long))

	filename, checksum := createTestBackupArchive(t, backupDir, 17, "old-format")
	repo := backupRepoMock()
	b := &models.Backup{ID: "old", Filename: filename, Checksum: checksum, Status: models.BackupStatusCompleted}
	repo.On("GetByID", mock.Anything, "old").Return(b, nil)
	svc := NewBackupService(db, repo, backupDir)
	svc.SetPosterDir(posterDir)

	res, err := svc.RestoreBackup(context.Background(), "old")
	require.NoError(t, err)
	require.Equal(t, models.RestoreStatusCompleted, res.Status, res.Error)
	assert.Equal(t, 0, res.PostersRestored)
	assert.Equal(t, "還原完成，資料庫已恢復", res.Message)
	info, err := os.Stat(filepath.Join(posterDir, "m1.jpg"))
	require.NoError(t, err)
	assert.True(t, info.ModTime().Equal(long), "an old-format backup must not touch the poster folder")
}

func TestBackupService_Restore_PosterFailureKeepsTheRestoredDatabase(t *testing.T) {
	svc, posterDir, b, _ := restoreFixture(t)
	require.NoError(t, os.Chmod(posterDir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(posterDir, 0o755) })

	res, err := svc.RestoreBackup(context.Background(), b.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RestoreStatusCompleted, res.Status, "the database is already restored")
	assert.Equal(t, 2, res.PostersFailed)
	assert.Equal(t, "還原完成，資料庫已恢復；有 2 張上傳的海報沒放回，原因見系統日誌", res.Message)
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// A posters/ entry that climbs out of the extract folder is refused, and a
// poster entry with no preceding directory entry still extracts.
func TestBackupService_ExtractTarGz_PosterEntries(t *testing.T) {
	db, _ := createTestDB(t)
	defer db.Close()
	svc := NewBackupService(db, nil, t.TempDir())

	build := func(name, content string) string {
		p := filepath.Join(t.TempDir(), "a.tar.gz")
		f, err := os.Create(p)
		require.NoError(t, err)
		gz := gzip.NewWriter(f)
		tw := tar.NewWriter(gz)
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(content)), Mode: 0o644, Typeflag: tar.TypeReg}))
		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
		require.NoError(t, tw.Close())
		require.NoError(t, gz.Close())
		require.NoError(t, f.Close())
		return p
	}

	dest := t.TempDir()
	require.NoError(t, svc.extractTarGz(build("posters/m1.jpg", "P"), dest))
	got, err := os.ReadFile(filepath.Join(dest, "posters", "m1.jpg"))
	require.NoError(t, err)
	assert.Equal(t, "P", string(got))

	assert.Error(t, svc.extractTarGz(build("posters/../../escape.jpg", "X"), t.TempDir()))
}
