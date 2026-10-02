package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// exchangeRepo is an in-memory glossary for one scope.
type exchangeRepo struct {
	scopeRecordingRepo
	rows     []models.GlossaryTerm
	inserted []models.GlossaryTerm
}

func (r *exchangeRepo) ListByScope(context.Context, string) ([]models.GlossaryTerm, error) {
	return r.rows, nil
}
func (r *exchangeRepo) InsertIfAbsent(_ context.Context, t *models.GlossaryTerm) (bool, error) {
	for _, have := range r.rows {
		if strings.EqualFold(have.TermSrc, t.TermSrc) {
			return false, nil
		}
	}
	r.inserted = append(r.inserted, *t)
	r.rows = append(r.rows, *t)
	return true, nil
}

func newExchange(rows []models.GlossaryTerm, scope string) (*GlossaryExchangeService, *exchangeRepo) {
	repo := &exchangeRepo{rows: rows}
	svc := NewGlossaryExchangeService(repo, fixedScopeResolver{scope: scope},
		func(context.Context, string) string { return "Shadow and Bone" }, nil)
	svc.now = func() time.Time { return time.Date(2026, 10, 2, 3, 0, 0, 0, time.FixedZone("TPE", 8*3600)) }
	return svc, repo
}

func TestGlossaryExport_ShapeAndOrder(t *testing.T) {
	svc, _ := newExchange([]models.GlossaryTerm{
		{ID: "a", TermSrc: "Ravka", TermZh: "拉夫卡", Source: models.GlossarySourceOfficialSubtitle},
		{ID: "b", TermSrc: "alina", TermZh: "阿利娜", Source: models.GlossarySourceMetadata, Confirmed: true},
		{ID: "c", TermSrc: "Grisha", TermZh: "格里沙", Language: "ja"}, // other language: not in this file
	}, "tmdb:tv:75006")
	f, err := svc.Export(context.Background(), "s1")
	require.NoError(t, err)
	assert.Equal(t, GlossaryExportFormat, f.Format)
	assert.Equal(t, 1, f.Version)
	assert.Equal(t, "tmdb:tv:75006", f.Scope)
	assert.Equal(t, "Shadow and Bone", f.Title)
	assert.Equal(t, models.GlossaryDefaultLanguage, f.Language)
	assert.Equal(t, time.UTC, f.ExportedAt.Location(), "exported_at is UTC (Rule 8)")
	assert.Equal(t, []GlossaryExportTerm{
		{TermSrc: "alina", TermZh: "阿利娜", Source: "metadata", Confirmed: true},
		{TermSrc: "Ravka", TermZh: "拉夫卡", Source: "official_subtitle"},
	}, f.Terms, "sorted case-insensitively, default language only")
}

func TestGlossaryExport_LocalScopeIsNotShareable(t *testing.T) {
	svc, _ := newExchange(nil, "local:s1")
	_, err := svc.Export(context.Background(), "s1")
	assert.ErrorIs(t, err, ErrGlossaryNotShareable)
}

func file(scope string, terms ...GlossaryExportTerm) *GlossaryExport {
	return &GlossaryExport{Format: GlossaryExportFormat, Version: 1, Scope: scope, Terms: terms}
}

func TestGlossaryImport_NewSkippedConflict(t *testing.T) {
	svc, repo := newExchange([]models.GlossaryTerm{
		{ID: "mine-1", TermSrc: "Ravka", TermZh: "拉夫卡", Source: models.GlossarySourceSubtitle},
		{ID: "mine-2", TermSrc: "Darkling", TermZh: "闇之手", Source: models.GlossarySourceManual, Confirmed: true},
	}, "tmdb:tv:75006")
	res, err := svc.Import(context.Background(), "s1", file("TMDB:TV:75006",
		GlossaryExportTerm{TermSrc: "Grisha", TermZh: "格里沙", Source: "official_subtitle", Confirmed: true}, // new
		GlossaryExportTerm{TermSrc: "ravka", TermZh: "拉夫卡"},                                                // same rendering → skipped
		GlossaryExportTerm{TermSrc: "Darkling", TermZh: "黑暗之主"},                                            // differs → conflict
		GlossaryExportTerm{TermSrc: "Grisha", TermZh: "格里莎"},                                               // dup in file → skipped
		GlossaryExportTerm{TermSrc: "  ", TermZh: "空"},                                                     // blank → skipped
	))
	require.NoError(t, err)
	assert.Equal(t, 1, res.Imported)
	assert.Equal(t, 3, res.Skipped)
	require.Len(t, res.Conflicts, 1)
	assert.Equal(t, GlossaryImportConflict{ID: "mine-2", TermSrc: "Darkling", Mine: "闇之手", Theirs: "黑暗之主", MineSource: "manual", MineConfirmed: true}, res.Conflicts[0])
	assert.Equal(t, "Shadow and Bone", res.Title)

	require.Len(t, repo.inserted, 1)
	got := repo.inserted[0]
	assert.Equal(t, models.GlossarySourceCommunity, got.Source, "imported terms are community, whatever the file says")
	assert.False(t, got.Confirmed, "and unconfirmed: the user approves them like any machine term")
	assert.Equal(t, "tmdb:tv:75006", got.Scope)
	assert.Equal(t, "s1", got.MediaID)
}

func TestGlossaryImport_Refusals(t *testing.T) {
	svc, repo := newExchange(nil, "tmdb:tv:75006")
	_, err := svc.Import(context.Background(), "s1", file("tmdb:tv:1", GlossaryExportTerm{TermSrc: "A", TermZh: "甲"}))
	assert.ErrorIs(t, err, ErrGlossaryScopeMismatch)

	bad := []*GlossaryExport{
		nil,
		{Format: "something-else", Version: 1, Scope: "tmdb:tv:75006"},
		{Format: GlossaryExportFormat, Version: 2, Scope: "tmdb:tv:75006"},
		{Format: GlossaryExportFormat, Version: 1},
		{Format: GlossaryExportFormat, Version: 1, Scope: "tmdb:tv:75006", Terms: []GlossaryExportTerm{{TermSrc: strings.Repeat("x", 201), TermZh: "甲"}}},
		{Format: GlossaryExportFormat, Version: 1, Scope: "tmdb:tv:75006", Terms: make([]GlossaryExportTerm, glossaryImportMaxTerms+1)},
	}
	for i, f := range bad {
		_, err := svc.Import(context.Background(), "s1", f)
		assert.ErrorIs(t, err, ErrGlossaryImportInvalid, "case %d", i)
	}
	assert.Empty(t, repo.inserted, "a refused file writes nothing")

	local, _ := newExchange(nil, "local:s1")
	_, err = local.Import(context.Background(), "s1", file("local:s1", GlossaryExportTerm{TermSrc: "A", TermZh: "甲"}))
	assert.ErrorIs(t, err, ErrGlossaryNotShareable)

	failing := &GlossaryExchangeService{repo: &exchangeRepo{}, scopes: fixedScopeResolver{err: errors.New("db down")}, now: time.Now, logger: newExchangeLogger()}
	_, err = failing.Export(context.Background(), "s1")
	assert.EqualError(t, err, "db down")
}

func newExchangeLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
