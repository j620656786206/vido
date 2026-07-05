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

// setupGlossaryDB applies the REAL migration chain (incl. 028) so the test
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

	term := &models.GlossaryTerm{MediaID: "m1", TermSrc: "Demogorgon", TermZh: "魔王獸", Source: models.GlossarySourceSubtitle}
	require.NoError(t, repo.Upsert(ctx, term))
	assert.NotEmpty(t, term.ID)
	assert.Equal(t, models.GlossaryDefaultLanguage, term.Language, "language defaults to zh-Hant")

	terms, err := repo.ListByMedia(ctx, "m1")
	require.NoError(t, err)
	require.Len(t, terms, 1)
	assert.Equal(t, "魔王獸", terms[0].TermZh)
	assert.False(t, terms[0].Confirmed)
}

func TestGlossaryRepository_UpsertConflictUpdatesInPlace(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	first := &models.GlossaryTerm{MediaID: "m1", TermSrc: "The Deep", TermZh: "深海怪物", Source: models.GlossarySourceSubtitle}
	require.NoError(t, repo.Upsert(ctx, first))

	// Re-mine the same term with a corrected rendering — must UPSERT, not duplicate.
	second := &models.GlossaryTerm{MediaID: "m1", TermSrc: "The Deep", TermZh: "深海", Source: models.GlossarySourceManual, Confirmed: true}
	require.NoError(t, repo.Upsert(ctx, second))

	terms, err := repo.ListByMedia(ctx, "m1")
	require.NoError(t, err)
	require.Len(t, terms, 1, "conflict on (media,term,language) must update in place")
	assert.Equal(t, "深海", terms[0].TermZh)
	assert.Equal(t, models.GlossarySourceManual, terms[0].Source)
	assert.True(t, terms[0].Confirmed)
}

func TestGlossaryRepository_LookupByMedia(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "m1", TermSrc: "Vecna", TermZh: "維克那", Confirmed: true}))
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "m1", TermSrc: "Mind Flayer", TermZh: "奪心魔", Confirmed: false}))
	require.NoError(t, repo.Upsert(ctx, &models.GlossaryTerm{MediaID: "m2", TermSrc: "Other", TermZh: "別的", Confirmed: true}))

	all, err := repo.LookupByMedia(ctx, "m1", false)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"Vecna": "維克那", "Mind Flayer": "奪心魔"}, all)

	confirmed, err := repo.LookupByMedia(ctx, "m1", true)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"Vecna": "維克那"}, confirmed, "confirmedOnly filters unconfirmed terms")
}

func TestGlossaryRepository_UpdateConfirmDelete(t *testing.T) {
	repo := NewGlossaryRepository(setupGlossaryDB(t))
	ctx := context.Background()

	term := &models.GlossaryTerm{MediaID: "m1", TermSrc: "Eleven", TermZh: "11", Confirmed: false}
	require.NoError(t, repo.Upsert(ctx, term))

	_, err := repo.Update(ctx, term.ID, "十一", true)
	require.NoError(t, err)
	terms, _ := repo.ListByMedia(ctx, "m1")
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
