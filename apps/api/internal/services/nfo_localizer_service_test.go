package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func sampleMovie(t *testing.T, dir string) models.Movie {
	t.Helper()
	return models.Movie{
		ID:            "42",
		Title:         "Inception",
		OriginalTitle: models.NewNullString("Inception"),
		ReleaseDate:   "2010",
		Overview:      models.NewNullString("A thief who steals corporate secrets through dream-sharing."),
		Genres:        []string{"Science Fiction", "Action"},
		VoteAverage:   models.NewNullFloat64(8.4),
		TMDbID:        models.NewNullInt64(27205),
		CreditsJSON:   models.NewNullString(`{"cast":[{"name":"Leonardo DiCaprio","character":"Dom Cobb"}],"crew":[{"name":"Christopher Nolan","job":"Director"}]}`),
		FilePath:      models.NewNullString(filepath.Join(dir, "Inception (2010).mkv")),
	}
}

// glossaryReturningStub returns fixed terms for LookupByMedia.
type glossaryReturningStub struct{ terms map[string]string }

func (g *glossaryReturningStub) Upsert(ctx context.Context, t *models.GlossaryTerm) error { return nil }
func (g *glossaryReturningStub) ListByMedia(ctx context.Context, mediaID string) ([]models.GlossaryTerm, error) {
	return nil, nil
}
func (g *glossaryReturningStub) LookupByMedia(ctx context.Context, mediaID string, confirmedOnly bool) (map[string]string, error) {
	return g.terms, nil
}
func (g *glossaryReturningStub) Update(ctx context.Context, id, termZh string, confirmed bool) (time.Time, error) {
	return time.Time{}, nil
}
func (g *glossaryReturningStub) Confirm(ctx context.Context, id string) (time.Time, error) {
	return time.Time{}, nil
}
func (g *glossaryReturningStub) ConfirmAll(ctx context.Context, mediaID string) (int64, error) {
	return 0, nil
}
func (g *glossaryReturningStub) Delete(ctx context.Context, id string) error { return nil }

func newLocalizer(t *testing.T, completerResp string, terms map[string]string) *NFOLocalizerService {
	t.Helper()
	completer := &mockTranslationCompleter{response: completerResp}
	svc := NewNFOLocalizerService(NewTranslationService(completer, nil), &glossaryReturningStub{terms: terms}, nil)
	require.NotNil(t, svc)
	return svc
}

func TestNFOLocalizer_NoOriginal_WritesFilenameSlot(t *testing.T) {
	dir := t.TempDir()
	movie := sampleMovie(t, dir)
	// 5 fields (title, plot, 2 genres, 1 role) → index-prefixed response.
	resp := "[1] 全面啟動\n[2] 一名盜賊竊取企業機密\n[3] 科幻\n[4] 動作\n[5] 唐姆·柯布"
	svc := newLocalizer(t, resp, map[string]string{"Dom Cobb": "唐姆·柯布"})

	res, err := svc.LocalizeMovieNFO(context.Background(), movie)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "Inception (2010).nfo"), res.Path)
	assert.False(t, res.Replaced)

	out, _ := os.ReadFile(res.Path)
	body := string(out)
	assert.Contains(t, body, "<title>全面啟動</title>")
	assert.Contains(t, body, "一名盜賊竊取企業機密")
	assert.Contains(t, body, "<genre>科幻</genre>")
	assert.Contains(t, body, "唐姆·柯布")
	// Preserved fields: original title, year, uniqueid, person name.
	assert.Contains(t, body, "<originaltitle>Inception</originaltitle>")
	assert.Contains(t, body, "<year>2010</year>")
	assert.Contains(t, body, `type="tmdb"`)
	assert.Contains(t, body, "Leonardo DiCaprio")
}

func TestNFOLocalizer_OriginalAtFilenameSlot_WritesMovieNfoAdditive(t *testing.T) {
	dir := t.TempDir()
	movie := sampleMovie(t, dir)
	orig := filepath.Join(dir, "Inception (2010).nfo")
	require.NoError(t, os.WriteFile(orig, []byte("<movie><title>Inception</title></movie>"), 0o644))

	svc := newLocalizer(t, "[1] 全面啟動\n[2] 劇情\n[3] 科幻\n[4] 動作\n[5] 柯布", nil)
	res, err := svc.LocalizeMovieNFO(context.Background(), movie)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "movie.nfo"), res.Path, "free-slot: original at <basename>.nfo → write movie.nfo")

	// Original untouched.
	origContent, _ := os.ReadFile(orig)
	assert.Equal(t, "<movie><title>Inception</title></movie>", string(origContent))
}

func TestNFOLocalizer_OriginalAtMovieNfo_WritesFilenameSlotAdditive(t *testing.T) {
	dir := t.TempDir()
	movie := sampleMovie(t, dir)
	orig := filepath.Join(dir, "movie.nfo")
	require.NoError(t, os.WriteFile(orig, []byte("ORIG"), 0o644))

	svc := newLocalizer(t, "[1] 全面啟動\n[2] 劇情\n[3] 科幻\n[4] 動作\n[5] 柯布", nil)
	res, err := svc.LocalizeMovieNFO(context.Background(), movie)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "Inception (2010).nfo"), res.Path)

	origContent, _ := os.ReadFile(orig)
	assert.Equal(t, "ORIG", string(origContent), "movie.nfo original preserved")
}

func TestNFOLocalizer_BothSlotsOccupied_BackupAndReplace(t *testing.T) {
	dir := t.TempDir()
	movie := sampleMovie(t, dir)
	fnSlot := filepath.Join(dir, "Inception (2010).nfo")
	require.NoError(t, os.WriteFile(fnSlot, []byte("ORIGINAL-FN"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "movie.nfo"), []byte("ORIGINAL-MOVIE"), 0o644))

	svc := newLocalizer(t, "[1] 全面啟動\n[2] 劇情\n[3] 科幻\n[4] 動作\n[5] 柯布", nil)
	res, err := svc.LocalizeMovieNFO(context.Background(), movie)
	require.NoError(t, err)
	assert.True(t, res.Replaced)
	assert.Equal(t, fnSlot, res.Path)
	assert.Equal(t, fnSlot+".orig", res.BackupPath)

	// The original was preserved in the .orig backup.
	backup, _ := os.ReadFile(fnSlot + ".orig")
	assert.Equal(t, "ORIGINAL-FN", string(backup), "original must survive in the .orig backup")
	// The slot now holds the zh-TW nfo.
	newContent, _ := os.ReadFile(fnSlot)
	assert.Contains(t, string(newContent), "全面啟動")
}

func TestNFOLocalizer_GlossaryInjectedIntoPrompt(t *testing.T) {
	dir := t.TempDir()
	movie := sampleMovie(t, dir)
	completer := &mockTranslationCompleter{response: "[1] 全面啟動\n[2] 劇情\n[3] 科幻\n[4] 動作\n[5] 唐姆·柯布"}
	svc := NewNFOLocalizerService(NewTranslationService(completer, nil), &glossaryReturningStub{terms: map[string]string{"Dom Cobb": "唐姆·柯布"}}, nil)

	_, err := svc.LocalizeMovieNFO(context.Background(), movie)
	require.NoError(t, err)
	require.Len(t, completer.calls, 1)
	assert.Contains(t, completer.calls[0].UserPrompt, "Dom Cobb → 唐姆·柯布")
	// All localizable fields batched in one request.
	assert.True(t, strings.Contains(completer.calls[0].UserPrompt, "Inception"))
}

func TestNFOLocalizer_NilWhenTranslationUnavailable(t *testing.T) {
	assert.Nil(t, NewNFOLocalizerService(nil, nil, nil))
}
