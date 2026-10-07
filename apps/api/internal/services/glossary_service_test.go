package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// scopeRecordingRepo captures which SCOPE each repository call was made with.
type scopeRecordingRepo struct {
	lastUpdateConfirmed *bool
	listed, confirmed   []string
	upserted            []models.GlossaryTerm
}

func (r *scopeRecordingRepo) Upsert(_ context.Context, t *models.GlossaryTerm) error {
	r.upserted = append(r.upserted, *t)
	return nil
}
func (r *scopeRecordingRepo) InsertIfAbsent(context.Context, *models.GlossaryTerm) (bool, error) {
	return false, nil
}
func (r *scopeRecordingRepo) ListByScope(_ context.Context, scope string) ([]models.GlossaryTerm, error) {
	r.listed = append(r.listed, scope)
	return nil, nil
}
func (r *scopeRecordingRepo) LookupByScope(context.Context, string, bool) (map[string]string, error) {
	return nil, nil
}
func (r *scopeRecordingRepo) Update(_ context.Context, _ string, _ string, confirmed bool) (time.Time, error) {
	r.lastUpdateConfirmed = &confirmed
	return time.Time{}, nil
}
func (r *scopeRecordingRepo) Confirm(context.Context, string) (time.Time, error) {
	return time.Time{}, nil
}
func (r *scopeRecordingRepo) ConfirmAllByScope(_ context.Context, scope string) (int64, error) {
	r.confirmed = append(r.confirmed, scope)
	return 3, nil
}
func (r *scopeRecordingRepo) Delete(context.Context, string) error { return nil }
func (r *scopeRecordingRepo) MigrateScope(context.Context, string, string) (int64, int64, error) {
	return 0, 0, nil
}
func (r *scopeRecordingRepo) IsScopeSeeded(context.Context, string) (bool, error) { return false, nil }
func (r *scopeRecordingRepo) MarkScopeSeeded(context.Context, string, int) error  { return nil }

type fixedScopeResolver struct {
	scope string
	err   error
}

func (f fixedScopeResolver) Resolve(context.Context, string) (string, error) { return f.scope, f.err }

// AC #4 / AC #5(d): the REST surface keeps speaking local ids; the service
// resolves them before every repository call.
func TestGlossaryService_ResolvesRouteIDToScope(t *testing.T) {
	repo := &scopeRecordingRepo{}
	svc := NewGlossaryService(repo, fixedScopeResolver{scope: "tmdb:tv:66732"})
	ctx := context.Background()

	_, err := svc.List(ctx, "series-42")
	require.NoError(t, err)
	n, err := svc.ConfirmAll(ctx, "series-42")
	require.NoError(t, err)
	assert.EqualValues(t, 3, n)
	require.NoError(t, svc.Add(ctx, &models.GlossaryTerm{MediaID: "series-42", TermSrc: "Vecna", TermZh: "維克那", Scope: "tmdb:tv:SMUGGLED"}))

	assert.Equal(t, []string{"tmdb:tv:66732"}, repo.listed)
	assert.Equal(t, []string{"tmdb:tv:66732"}, repo.confirmed)
	require.Len(t, repo.upserted, 1)
	assert.Equal(t, "tmdb:tv:66732", repo.upserted[0].Scope, "the route id decides the scope — a body scope is overwritten")
	assert.Equal(t, "series-42", repo.upserted[0].MediaID)
	assert.Equal(t, models.GlossarySourceManual, repo.upserted[0].Source)
}

func TestGlossaryService_NoResolverKeysTheLocalDrawer(t *testing.T) {
	repo := &scopeRecordingRepo{}
	svc := NewGlossaryService(repo, nil)
	_, err := svc.List(context.Background(), "series-42")
	require.NoError(t, err)
	assert.Equal(t, []string{"local:series-42"}, repo.listed)
}

func TestGlossaryService_ResolverErrorStopsTheCall(t *testing.T) {
	repo := &scopeRecordingRepo{}
	svc := NewGlossaryService(repo, fixedScopeResolver{err: errors.New("db down")})
	_, err := svc.List(context.Background(), "series-42")
	require.Error(t, err)
	assert.Empty(t, repo.listed, "no repository call on a resolve failure — the UI gets the error, not an empty drawer")
}

// ⚖️ Alexyu 2026-10-02: an edit always confirms, whatever the body says.
func TestGlossaryService_EditAlwaysConfirms(t *testing.T) {
	repo := &scopeRecordingRepo{}
	svc := NewGlossaryService(repo, fixedScopeResolver{scope: "tmdb:tv:1"})
	require.NoError(t, svc.Edit(context.Background(), "42", "g1", "魔神獸", false))
	require.NotNil(t, repo.lastUpdateConfirmed)
	assert.True(t, *repo.lastUpdateConfirmed, "a reviewed-and-rewritten term is confirmed")
}

// drawerAwareRepo adds the optional per-season surface to the inert fake.
type drawerAwareRepo struct {
	scopeRecordingRepo
	base    []models.GlossaryTerm
	drawers []models.GlossaryTerm
	cleared []string
}

func (r *drawerAwareRepo) ListByScope(context.Context, string) ([]models.GlossaryTerm, error) {
	return r.base, nil
}
func (r *drawerAwareRepo) ListSeasonDrawers(context.Context, string) ([]models.GlossaryTerm, error) {
	return r.drawers, nil
}
func (r *drawerAwareRepo) ClearSeasonDrawersOfTerm(_ context.Context, id string) (int64, error) {
	r.cleared = append(r.cleared, id)
	return 1, nil
}
func (r *drawerAwareRepo) DeleteSeasonDrawers(_ context.Context, scope string) (int64, error) {
	r.cleared = append(r.cleared, "ALL:"+scope)
	return 2, nil
}

// disc-2026-10-glossary-season-scope-b: the panel list interleaves each term's
// season drawers right after its show-wide row; an edit / confirm / delete of
// the show-wide row clears that term's drawers.
func TestGlossaryService_ListInterleavesSeasonDrawers_AndWritesClearThem(t *testing.T) {
	one, two := 1, 2
	repo := &drawerAwareRepo{
		base: []models.GlossaryTerm{{ID: "h", TermSrc: "Haniwa", TermZh: "哈妮娃"}, {ID: "j", TermSrc: "Jerlamarel", TermZh: "傑拉馬瑞"}},
		drawers: []models.GlossaryTerm{
			{ID: "j1", TermSrc: "Jerlamarel", TermZh: "謝拉馬威", Scope: "tmdb:tv:80752:s1", Season: &one},
			{ID: "j2", TermSrc: "Jerlamarel", TermZh: "傑拉馬瑞", Scope: "tmdb:tv:80752:s2", Season: &two},
		},
	}
	svc := NewGlossaryService(repo, &fakeScopeResolver{scope: "tmdb:tv:80752"})
	got, err := svc.List(context.Background(), "series-1")
	require.NoError(t, err)
	var ids []string
	for _, g := range got {
		ids = append(ids, g.ID)
	}
	assert.Equal(t, []string{"h", "j", "j1", "j2"}, ids)

	require.NoError(t, svc.Edit(context.Background(), "series-1", "j", "傑拉瑪瑞爾", true))
	require.NoError(t, svc.Confirm(context.Background(), "series-1", "h"))
	require.NoError(t, svc.Delete(context.Background(), "series-1", "j"))
	assert.Equal(t, []string{"j", "h", "j"}, repo.cleared)

	// CR 1: 全部確認 makes every show-wide row the user's word → all drawers go.
	_, err = svc.ConfirmAll(context.Background(), "series-1")
	require.NoError(t, err)
	assert.Equal(t, "ALL:tmdb:tv:80752", repo.cleared[len(repo.cleared)-1])
}
