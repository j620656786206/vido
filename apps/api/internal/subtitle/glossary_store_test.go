package subtitle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// captureGlossaryRepo records InsertIfAbsent traffic; other methods are inert.
type captureGlossaryRepo struct {
	terms     []models.GlossaryTerm
	lookedUp  []string
	failOn    string
	preseeded map[string]bool // term_src → exists already (DO NOTHING)
}

// scopeStub is a GlossaryScopeResolver that answers one fixed scope and
// records what it was asked.
type scopeStub struct {
	scope string
	err   error
	asked []string
}

func (s *scopeStub) Resolve(_ context.Context, mediaID string) (string, error) {
	s.asked = append(s.asked, mediaID)
	return s.scope, s.err
}

func (c *captureGlossaryRepo) Upsert(context.Context, *models.GlossaryTerm) error { return nil }
func (c *captureGlossaryRepo) ListByScope(context.Context, string) ([]models.GlossaryTerm, error) {
	return nil, nil
}
func (c *captureGlossaryRepo) LookupByScope(_ context.Context, scope string, _ bool) (map[string]string, error) {
	c.lookedUp = append(c.lookedUp, scope)
	return map[string]string{"Vecna": "維克那"}, nil
}
func (c *captureGlossaryRepo) MigrateScope(context.Context, string, string) (int64, int64, error) {
	return 0, 0, nil
}
func (c *captureGlossaryRepo) Update(context.Context, string, string, bool) (time.Time, error) {
	return time.Time{}, nil
}
func (c *captureGlossaryRepo) Confirm(context.Context, string) (time.Time, error) {
	return time.Time{}, nil
}
func (c *captureGlossaryRepo) ConfirmAllByScope(context.Context, string) (int64, error) {
	return 0, nil
}
func (c *captureGlossaryRepo) Delete(context.Context, string) error { return nil }

func (c *captureGlossaryRepo) InsertIfAbsent(_ context.Context, term *models.GlossaryTerm) (bool, error) {
	if term.TermSrc == c.failOn {
		return false, errors.New("disk full")
	}
	c.terms = append(c.terms, *term)
	return !c.preseeded[term.TermSrc], nil
}

// TestGlossaryStoreRepository_InsertNewValueChain pins the harvest write's
// fixed values (sub-5-5 AC #4): source='subtitle', confirmed=0 — the value
// chain that routes every harvested term into the existing F6 review flow.
func TestGlossaryStoreRepository_InsertNewValueChain(t *testing.T) {
	repo := &captureGlossaryRepo{preseeded: map[string]bool{"Vecna": true}}
	store := NewGlossaryStoreRepository(repo, nil)

	inserted, err := store.InsertNew(context.Background(), "series-42", map[string]string{
		"Demogorgon": "魔王獸",
		"Vecna":      "威克納", // already in the glossary — DO NOTHING, not counted
	})
	require.NoError(t, err)
	assert.Equal(t, 1, inserted, "deduped conflicts are excluded from the harvested count")

	require.Len(t, repo.terms, 2)
	for _, term := range repo.terms {
		assert.Equal(t, "series-42", term.MediaID)
		// No resolver wired → the pre-sub-7-1 key under its new name.
		assert.Equal(t, "local:series-42", term.Scope)
		assert.Equal(t, models.GlossarySourceSubtitle, term.Source)
		assert.False(t, term.Confirmed, "harvested terms are never silently trusted")
	}
}

func TestGlossaryStoreRepository_InsertNewIsBestEffort(t *testing.T) {
	repo := &captureGlossaryRepo{failOn: "Demogorgon"}
	store := NewGlossaryStoreRepository(repo, nil)

	inserted, err := store.InsertNew(context.Background(), "series-42", map[string]string{
		"Demogorgon": "魔王獸",
		"Eleven":     "十一",
	})
	require.Error(t, err, "the per-term failure surfaces for the caller's fail-soft log")
	assert.Equal(t, 1, inserted, "the failing term is skipped; the rest still land")
}

func TestGlossaryStoreRepository_LookupFeedsAllTerms(t *testing.T) {
	store := NewGlossaryStoreRepository(&captureGlossaryRepo{}, nil)
	got, err := store.Lookup(context.Background(), "series-42")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"Vecna": "維克那"}, got)
}

// sub-7-1 AC #2/#5(d): the adapter is where the pipeline's local show key
// becomes a scope — both the feed read and the harvest write go through it.
func TestGlossaryStoreRepository_ResolvesKeyToScopeForFeedAndHarvest(t *testing.T) {
	repo := &captureGlossaryRepo{}
	scopes := &scopeStub{scope: "tmdb:tv:66732"}
	store := NewGlossaryStoreRepository(repo, scopes)

	_, err := store.Lookup(context.Background(), "series-42")
	require.NoError(t, err)
	assert.Equal(t, []string{"tmdb:tv:66732"}, repo.lookedUp, "the feed reads the RESOLVED scope")

	_, err = store.InsertNew(context.Background(), "series-42", map[string]string{"Vecna": "維克那"})
	require.NoError(t, err)
	require.Len(t, repo.terms, 1)
	assert.Equal(t, "tmdb:tv:66732", repo.terms[0].Scope, "the harvest writes into the RESOLVED scope")
	assert.Equal(t, "series-42", repo.terms[0].MediaID, "the local id stays as the audit column")
	assert.Equal(t, []string{"series-42", "series-42"}, scopes.asked)
}

func TestGlossaryStoreRepository_ResolverErrorSurfacesToTheFailSoftCaller(t *testing.T) {
	store := NewGlossaryStoreRepository(&captureGlossaryRepo{}, &scopeStub{err: errors.New("db down")})
	_, err := store.Lookup(context.Background(), "series-42")
	require.Error(t, err, "the pipeline's feedGlossary logs and translates without a glossary")
}
