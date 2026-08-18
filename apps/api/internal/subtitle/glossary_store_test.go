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
	failOn    string
	preseeded map[string]bool // term_src → exists already (DO NOTHING)
}

func (c *captureGlossaryRepo) Upsert(context.Context, *models.GlossaryTerm) error { return nil }
func (c *captureGlossaryRepo) ListByMedia(context.Context, string) ([]models.GlossaryTerm, error) {
	return nil, nil
}
func (c *captureGlossaryRepo) LookupByMedia(context.Context, string, bool) (map[string]string, error) {
	return map[string]string{"Vecna": "維克那"}, nil
}
func (c *captureGlossaryRepo) Update(context.Context, string, string, bool) (time.Time, error) {
	return time.Time{}, nil
}
func (c *captureGlossaryRepo) Confirm(context.Context, string) (time.Time, error) {
	return time.Time{}, nil
}
func (c *captureGlossaryRepo) ConfirmAll(context.Context, string) (int64, error) { return 0, nil }
func (c *captureGlossaryRepo) Delete(context.Context, string) error              { return nil }

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
	store := NewGlossaryStoreRepository(repo)

	inserted, err := store.InsertNew(context.Background(), "series-42", map[string]string{
		"Demogorgon": "魔王獸",
		"Vecna":      "威克納", // already in the glossary — DO NOTHING, not counted
	})
	require.NoError(t, err)
	assert.Equal(t, 1, inserted, "deduped conflicts are excluded from the harvested count")

	require.Len(t, repo.terms, 2)
	for _, term := range repo.terms {
		assert.Equal(t, "series-42", term.MediaID)
		assert.Equal(t, models.GlossarySourceSubtitle, term.Source)
		assert.False(t, term.Confirmed, "harvested terms are never silently trusted")
	}
}

func TestGlossaryStoreRepository_InsertNewIsBestEffort(t *testing.T) {
	repo := &captureGlossaryRepo{failOn: "Demogorgon"}
	store := NewGlossaryStoreRepository(repo)

	inserted, err := store.InsertNew(context.Background(), "series-42", map[string]string{
		"Demogorgon": "魔王獸",
		"Eleven":     "十一",
	})
	require.Error(t, err, "the per-term failure surfaces for the caller's fail-soft log")
	assert.Equal(t, 1, inserted, "the failing term is skipped; the rest still land")
}

func TestGlossaryStoreRepository_LookupFeedsAllTerms(t *testing.T) {
	store := NewGlossaryStoreRepository(&captureGlossaryRepo{})
	got, err := store.Lookup(context.Background(), "series-42")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"Vecna": "維克那"}, got)
}
