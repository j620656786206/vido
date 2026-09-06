package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	_ "modernc.org/sqlite"
)

// setupGlossaryDB applies the REAL migration chain (incl. 028 + 036) so the test
// tracks the shipped schema (Rule 15 — no hand-copied schema literals).
func setupGlossaryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

func TestGlossaryRepository_UpsertAndList(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	term := &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "Demogorgon", TermZh: "魔王獸", Source: models.GlossarySourceSubtitle}
	require.NoError(t, repo.Upsert(ctx, term))
	assert.NotEmpty(t, term.ID)
	assert.Equal(t, models.GlossaryDefaultLanguage, term.Language, "language defaults to zh-Hant")

	terms, err := repo.ListByScope(ctx, "local:m1")
	require.NoError(t, err)
	require.Len(t, terms, 1)
	assert.Equal(t, "魔王獸", terms[0].TermZh)
	assert.False(t, terms[0].Confirmed)
}

func TestGlossaryRepository_UpsertConflictUpdatesInPlace(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	first := &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "The Deep", TermZh: "深海怪物", Source: models.GlossarySourceSubtitle}
	require.NoError(t, repo.Upsert(ctx, first))

	// Re-mine the same term with a corrected rendering — must UPSERT, not duplicate.
	second := &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "The Deep", TermZh: "深海", Source: models.GlossarySourceManual, Confirmed: true}
	require.NoError(t, repo.Upsert(ctx, second))

	terms, err := repo.ListByScope(ctx, "local:m1")
	require.NoError(t, err)
	require.Len(t, terms, 1, "conflict on (media,term,language) must update in place")
	assert.Equal(t, "深海", terms[0].TermZh)
	assert.Equal(t, models.GlossarySourceManual, terms[0].Source)
	assert.True(t, terms[0].Confirmed)
}

// --- sub-5-5 AC #4 (red line 2): harvest must NEVER overwrite an existing term ---

func TestGlossaryRepository_InsertIfAbsent_NeverTouchesExistingTerm(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	// A user-corrected, CONFIRMED manual term — the exact row Upsert's DO UPDATE
	// would clobber back to the machine rendering and un-confirm.
	existing := &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "Vecna", TermZh: "維克那",
		Source: models.GlossarySourceManual, Confirmed: true}
	require.NoError(t, repo.Upsert(ctx, existing))
	before, err := repo.ListByScope(ctx, "local:m1")
	require.NoError(t, err)
	require.Len(t, before, 1)

	// Harvest re-mines the same term with a DIFFERENT machine rendering.
	inserted, err := repo.InsertIfAbsent(ctx, &models.GlossaryTerm{
		MediaID: "m1", Scope: "local:m1", TermSrc: "Vecna", TermZh: "威克納", Source: models.GlossarySourceSubtitle})
	require.NoError(t, err)
	assert.False(t, inserted, "conflicting harvest must be a DO NOTHING, not an update")

	after, err := repo.ListByScope(ctx, "local:m1")
	require.NoError(t, err)
	require.Len(t, after, 1)
	assert.Equal(t, before[0], after[0],
		"red line 2: not a single field of the existing term may change — id, rendering, source, confirmed, timestamps")
}

func TestGlossaryRepository_InsertIfAbsent_NewTermLandsUnconfirmed(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	inserted, err := repo.InsertIfAbsent(ctx, &models.GlossaryTerm{
		MediaID: "m1", Scope: "local:m1", TermSrc: "Demogorgon", TermZh: "魔王獸", Source: models.GlossarySourceSubtitle})
	require.NoError(t, err)
	assert.True(t, inserted)

	terms, err := repo.ListByScope(ctx, "local:m1")
	require.NoError(t, err)
	require.Len(t, terms, 1)
	assert.Equal(t, "魔王獸", terms[0].TermZh)
	assert.Equal(t, models.GlossarySourceSubtitle, terms[0].Source)
	assert.False(t, terms[0].Confirmed, "harvested terms enter the existing F6 review flow unconfirmed")
	assert.Equal(t, models.GlossaryDefaultLanguage, terms[0].Language)
}

func TestGlossaryRepository_InsertIfAbsent_ValidatesLikeUpsert(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	_, err := repo.InsertIfAbsent(context.Background(), &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "", TermZh: "x"})
	require.Error(t, err)
}

func TestGlossaryRepository_LookupByMedia(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "Vecna", TermZh: "維克那", Confirmed: true}))
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "Mind Flayer", TermZh: "奪心魔", Confirmed: false}))
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "m2", Scope: "local:m2", TermSrc: "Other", TermZh: "別的", Confirmed: true}))

	all, err := repo.LookupByScope(ctx, "local:m1", false)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"Vecna": "維克那", "Mind Flayer": "奪心魔"}, all)

	confirmed, err := repo.LookupByScope(ctx, "local:m1", true)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"Vecna": "維克那"}, confirmed, "confirmedOnly filters unconfirmed terms")
}

func TestGlossaryRepository_UpdateConfirmDelete(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	term := &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "Eleven", TermZh: "11", Confirmed: false}
	require.NoError(t, repo.Upsert(ctx, term))

	_, err := repo.Update(ctx, term.ID, "十一", true)
	require.NoError(t, err)
	terms, _ := repo.ListByScope(ctx, "local:m1")
	require.Len(t, terms, 1)
	assert.Equal(t, "十一", terms[0].TermZh)
	assert.True(t, terms[0].Confirmed)

	// Confirm on an unknown id → not found.
	_, err = repo.Confirm(ctx, "nope")
	require.ErrorIs(t, err, ErrGlossaryTermNotFound)

	require.NoError(t, repo.Delete(ctx, term.ID))
	err = repo.Delete(ctx, term.ID)
	require.ErrorIs(t, err, ErrGlossaryTermNotFound, "second delete → not found")
}

func TestGlossaryRepository_UpsertValidation(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	err := repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "", TermSrc: "x", TermZh: "y"})
	require.Error(t, err)
	var ve *models.ValidationError
	assert.True(t, errors.As(err, &ve))
}

func TestGlossaryRepository_ConfirmAll(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "A", TermZh: "甲", Confirmed: false}))
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "B", TermZh: "乙", Confirmed: false}))
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "C", TermZh: "丙", Confirmed: true}))
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "m2", Scope: "local:m2", TermSrc: "X", TermZh: "叉", Confirmed: false}))

	n, err := repo.ConfirmAllByScope(ctx, "local:m1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), n, "only the 2 unconfirmed m1 terms flip")

	confirmed, _ := repo.LookupByScope(ctx, "local:m1", true)
	assert.Len(t, confirmed, 3, "all m1 terms now confirmed")

	// m2 untouched.
	m2, _ := repo.LookupByScope(ctx, "local:m2", true)
	assert.Len(t, m2, 0)
}

// ---------------------------------------------------------------------------
// sub-7-1: scope keying, NOCASE + trim normalisation, five sources, MigrateScope
// ---------------------------------------------------------------------------

func TestGlossaryRepository_ScopeIsTheKey_NotMediaID(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	// Two local copies of the same show (different media ids) share ONE drawer.
	a := &models.GlossaryTerm{MediaID: "copy-a", Scope: "tmdb:tv:66732", TermSrc: "Vecna", TermZh: "維克那"}
	b := &models.GlossaryTerm{MediaID: "copy-b", Scope: "tmdb:tv:66732", TermSrc: "Eleven", TermZh: "十一"}
	require.NoError(t, repo.Upsert(ctx, a))
	require.NoError(t, repo.Upsert(ctx, b))

	terms, err := repo.ListByScope(ctx, "tmdb:tv:66732")
	require.NoError(t, err)
	assert.Len(t, terms, 2, "both copies read the same shared drawer")
	// The audit column still says who wrote each row.
	assert.ElementsMatch(t, []string{"copy-a", "copy-b"}, []string{terms[0].MediaID, terms[1].MediaID})

	// The old media-id delegates only ever see the LOCAL drawer.
	old, err := repo.ListByMedia(ctx, "copy-a")
	require.NoError(t, err)
	assert.Empty(t, old, "Deprecated ByMedia = local:<id> only; shared terms are invisible to it")
}

func TestGlossaryRepository_TermSrcIsTrimmedAndCaseInsensitivelyUnique(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	first := &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "  Demogorgon ", TermZh: "魔王獸"}
	require.NoError(t, repo.Upsert(ctx, first))
	assert.Equal(t, "Demogorgon", first.TermSrc, "whitespace is trimmed on the way in")

	// Same term, different case: the NOCASE index says "same drawer, same term".
	ok, err := repo.InsertIfAbsent(ctx, &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "demogorgon", TermZh: "別的"})
	require.NoError(t, err)
	assert.False(t, ok, "case-variant of an existing term is a conflict, not a new row")

	terms, err := repo.ListByScope(ctx, "local:m1")
	require.NoError(t, err)
	require.Len(t, terms, 1)
	assert.Equal(t, "Demogorgon", terms[0].TermSrc, "the FIRST spelling is kept — case is not folded on the stored value")
	assert.Equal(t, "魔王獸", terms[0].TermZh)

	// Upsert with the case-variant updates that same row in place.
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "DEMOGORGON", TermZh: "魔王獸（新）", Confirmed: true}))
	terms, _ = repo.ListByScope(ctx, "local:m1")
	require.Len(t, terms, 1)
	assert.Equal(t, "魔王獸（新）", terms[0].TermZh)
	assert.Equal(t, "Demogorgon", terms[0].TermSrc, "upsert keeps the original spelling too")
}

func TestGlossaryRepository_AllFiveSourcesAreWritable(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()
	for i, src := range models.GlossarySources {
		term := &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "term-" + src, TermZh: "譯", Source: src}
		require.NoError(t, repo.Upsert(ctx, term), "source #%d %q must be accepted now that the CHECK is gone", i, src)
	}
	err := repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "m1", Scope: "local:m1", TermSrc: "x", TermZh: "y", Source: "bogus"})
	var verr *models.ValidationError
	require.ErrorAs(t, err, &verr, "an unknown source is rejected by the model, not by SQLite")
	assert.Equal(t, "source", verr.Field)
}

func TestGlossaryRepository_ScopeIsRequired(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	err := repo.Upsert(context.Background(), &models.GlossaryTerm{MediaID: "m1", TermSrc: "x", TermZh: "y"})
	var verr *models.ValidationError
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "scope", verr.Field)
}

func TestGlossaryRepository_MigrateScope_MovesLocalIntoSharedWithoutOverwriting(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	// Written before the TMDb match landed:
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "s1", Scope: "local:s1", TermSrc: "Vecna", TermZh: "本機版維克那"}))
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "s1", Scope: "local:s1", TermSrc: "Eleven", TermZh: "十一"}))
	// Already in the shared drawer (another copy, or a seed) — and CONFIRMED.
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "s9", Scope: "tmdb:tv:66732", TermSrc: "vecna", TermZh: "維克那", Confirmed: true}))

	moved, skipped, err := repo.MigrateScope(ctx, "local:s1", "tmdb:tv:66732")
	require.NoError(t, err)
	assert.EqualValues(t, 1, moved, "Eleven moves")
	assert.EqualValues(t, 1, skipped, "Vecna stays behind — the shared drawer already has it (NOCASE)")

	shared, err := repo.LookupByScope(ctx, "tmdb:tv:66732", false)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"vecna": "維克那", "Eleven": "十一"}, shared,
		"the confirmed shared rendering is NEVER overwritten by the stale local one")

	left, err := repo.ListByScope(ctx, "local:s1")
	require.NoError(t, err)
	require.Len(t, left, 1)
	assert.Equal(t, "Vecna", left[0].TermSrc)

	// A second move is a no-op (nothing left to move that would not collide).
	moved, skipped, err = repo.MigrateScope(ctx, "local:s1", "tmdb:tv:66732")
	require.NoError(t, err)
	assert.EqualValues(t, 0, moved)
	assert.EqualValues(t, 1, skipped)
}

func TestGlossaryRepository_MigrateScope_RejectsNonMoves(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	_, _, err := repo.MigrateScope(context.Background(), "local:s1", "local:s1")
	require.Error(t, err)
	_, _, err = repo.MigrateScope(context.Background(), "", "tmdb:tv:1")
	require.Error(t, err)
}
