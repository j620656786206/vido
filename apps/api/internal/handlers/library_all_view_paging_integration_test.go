package handlers

// Story disc-2026-10-library-all-view-past-100 AC #7 (handler leg, Rule 28
// real shape): the 「全部」 list, sorted by 評分, paged through the REAL
// handler → LibraryService → repositories on a migrated sqlite file holding
// 247 live titles (130 movies + 120 series, 3 soft-deleted), legacy `rating`
// NULL everywhere like production. Every title appears exactly once, the
// highest score is first, no-score titles are last.
//
// `go test ./internal/handlers -run TestLibraryAllView_RatingSortPagesThrough -v`
// prints the numbers quoted in the story's Completion Notes.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/database/dbtime"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/services"
)

type ratingWireItem struct {
	Type  string `json:"type"`
	Movie *struct {
		ID          string   `json:"id"`
		VoteAverage *float64 `json:"vote_average"`
	} `json:"movie"`
	Series *struct {
		ID          string   `json:"id"`
		VoteAverage *float64 `json:"vote_average"`
	} `json:"series"`
}

func (it ratingWireItem) key() (string, *float64) {
	if it.Movie != nil {
		return "movie:" + it.Movie.ID, it.Movie.VoteAverage
	}
	if it.Series != nil {
		return "series:" + it.Series.ID, it.Series.VoteAverage
	}
	return "", nil
}

func TestLibraryAllView_RatingSortPagesThrough(t *testing.T) {
	db := setupRouteCTestDB(t)
	ctx := context.Background()
	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	libSvc := services.NewLibraryService(movieRepo, seriesRepo, repository.NewEpisodeRepository(db), repository.NewLibraryRepository(db))
	router := setupLibraryTestRouter(NewLibraryHandler(libSvc))

	votes := []interface{}{6.0, 7.3, 7.7, 8.5, 9.5, nil}
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	seed := func(table string, i int) {
		id := fmt.Sprintf("%s-%03d", table[:1], i)
		if table == "movies" {
			require.NoError(t, movieRepo.Create(ctx, &models.Movie{ID: id, Title: fmt.Sprintf("Title %02d", i%37), ReleaseDate: "2020-01-01"}))
		} else {
			require.NoError(t, seriesRepo.Create(ctx, &models.Series{ID: id, Title: fmt.Sprintf("Title %02d", i%37), FirstAirDate: "2020-01-01"}))
		}
		_, err := db.ExecContext(ctx, `UPDATE `+table+` SET rating = NULL, vote_average = ?, created_at = ? WHERE id = ?`,
			votes[(i*5)%len(votes)], dbtime.Format(base.Add(time.Duration((i*7)%10)*time.Hour)), id)
		require.NoError(t, err)
	}
	for i := 0; i < 130; i++ {
		seed("movies", i)
	}
	for i := 0; i < 120; i++ {
		seed("series", i)
	}
	for _, q := range []string{
		`UPDATE movies SET is_removed = 1 WHERE id IN ('m-003', 'm-064')`,
		`UPDATE series SET is_removed = 1 WHERE id = 's-017'`,
	} {
		_, err := db.ExecContext(ctx, q)
		require.NoError(t, err)
	}

	type pageBody struct {
		Data struct {
			Items      []ratingWireItem `json:"items"`
			Page       int              `json:"page"`
			PageSize   int              `json:"page_size"`
			TotalItems int              `json:"total_items"`
			TotalPages int              `json:"total_pages"`
		} `json:"data"`
	}

	var keys []string
	var scores []*float64
	totalPages := 1
	for page := 1; page <= totalPages; page++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/library?sort_by=rating&sort_order=desc&page_size=36&page=%d", page), nil)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body pageBody
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Equal(t, 247, body.Data.TotalItems)
		require.Equal(t, 36, body.Data.PageSize)
		totalPages = body.Data.TotalPages
		for _, it := range body.Data.Items {
			k, v := it.key()
			keys = append(keys, k)
			scores = append(scores, v)
		}
	}

	unique := map[string]bool{}
	for _, k := range keys {
		unique[k] = true
	}
	fmtScore := func(v *float64) string {
		if v == nil {
			return "null"
		}
		return fmt.Sprintf("%.1f", *v)
	}
	var first5, last3 []string
	for _, v := range scores[:5] {
		first5 = append(first5, fmtScore(v))
	}
	for _, v := range scores[len(scores)-3:] {
		last3 = append(last3, fmtScore(v))
	}
	t.Logf("AC #7: total_pages=%d rows=%d unique=%d first5=%v last3=%v", totalPages, len(keys), len(unique), first5, last3)

	assert.Equal(t, 7, totalPages)
	assert.Len(t, keys, 247)
	assert.Len(t, unique, 247)
	assert.Equal(t, []string{"9.5", "9.5", "9.5", "9.5", "9.5"}, first5)
	assert.Equal(t, []string{"null", "null", "null"}, last3)
	// Never increasing; once the no-score titles start, nothing else follows.
	for i := 1; i < len(scores); i++ {
		prev, cur := scores[i-1], scores[i]
		if prev == nil {
			assert.Nil(t, cur, "row %d: a scored title after the no-score ones", i)
			continue
		}
		if cur != nil {
			assert.LessOrEqual(t, *cur, *prev, "row %d: score went up", i)
		}
	}
}
