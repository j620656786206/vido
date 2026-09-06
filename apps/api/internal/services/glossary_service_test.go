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
	listed, confirmed []string
	upserted          []models.GlossaryTerm
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
func (r *scopeRecordingRepo) Update(context.Context, string, string, bool) (time.Time, error) {
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
