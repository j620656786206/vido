package repository

import (
	"context"
	"testing"

	"github.com/vido/api/internal/models"
)

func TestFtsPrefixQuery(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"駭客", `"駭客"*`},
		{"駭客任務", `"駭客任務"*`},
		{"the matrix", `"the"* "matrix"*`},
		{`he said "hi"`, `"he"* "said"* """hi"""*`},
		{"  spaced   out  ", `"spaced"* "out"*`},
		{`AND OR NOT -`, `"AND"* "OR"* "NOT"* "-"*`},
	}
	for _, c := range cases {
		if got := ftsPrefixQuery(c.in); got != c.want {
			t.Errorf("ftsPrefixQuery(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// The testsprite-round1 TC092 chain: a PARTIAL zh-TW query must find the owned
// title. unicode61 keeps a CJK run as one token, so this only works through the
// quoted-prefix transformation.
func TestMovieFullTextSearch_CJKPartialQuery(t *testing.T) {
	db := setupTestDBWithFTS(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()
	for _, m := range []*models.Movie{
		{ID: "mv-matrix", Title: "駭客任務", OriginalTitle: models.NewNullString("The Matrix"), ReleaseDate: "1999-03-31", Genres: []string{"科幻"}},
		{ID: "mv-godfather", Title: "教父", ReleaseDate: "1972-03-14", Genres: []string{"犯罪"}},
	} {
		if err := repo.Create(ctx, m); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	got, _, err := repo.FullTextSearch(ctx, "駭客", NewListParams())
	if err != nil {
		t.Fatalf("partial CJK search errored: %v", err)
	}
	if len(got) != 1 || got[0].ID != "mv-matrix" {
		t.Fatalf("expected 駭客 to prefix-match 駭客任務, got %+v", got)
	}

	// English prefix also benefits.
	got, _, err = repo.FullTextSearch(ctx, "matr", NewListParams())
	if err != nil {
		t.Fatalf("english prefix search errored: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected matr to prefix-match The Matrix original title... got %d", len(got))
	}
}

// Raw FTS5 operators in user input must not raise fts5 syntax errors (they used
// to 500 the search endpoints).
func TestMovieFullTextSearch_OperatorInputIsInert(t *testing.T) {
	db := setupTestDBWithFTS(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()
	if err := repo.Create(ctx, &models.Movie{ID: "mv-1", Title: "教父", ReleaseDate: "1972-03-14", Genres: []string{"犯罪"}}); err != nil {
		t.Fatalf("create: %v", err)
	}

	for _, q := range []string{`教父 -`, `"unbalanced`, `a AND`, `(paren`} {
		if _, _, err := repo.FullTextSearch(ctx, q, NewListParams()); err != nil {
			t.Errorf("query %q should be inert, got error: %v", q, err)
		}
	}
}
