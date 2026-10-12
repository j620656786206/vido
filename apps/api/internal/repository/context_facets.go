package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/vido/api/internal/models"
)

// Context facets — production countries and genres, the two TMDb-details
// fields the subtitle pipeline reads as "what kind of title is this"
// (disc-2026-10-production-countries-never-filled).
//
// sub-7-2b (2026-09-30) made every TMDb match fetch the details and put both
// fields on the model, but the enrichment writer (UpdateEnrichedMetadata)
// never wrote production_countries, so on the NAS not one of 136 titles has a
// country — and the CN/HK/MO "keep the original wording" rule never fired.
// Rows matched before sub-7-2b also have empty genres, and nothing re-matches
// a matched row.
//
// Two writers, two ownership rules:
//   - UpdateProductionCountries: the match path. A fresh match is the
//     authority, so a value replaces the old one; NULL ("TMDb gave none this
//     time") never erases a stored answer.
//   - FillContextFacets: the background backfill. It only ever fills an empty
//     column, so it cannot overwrite a match or an edit that landed while it
//     was asking TMDb.
//
// Neither touches updated_at: these are derived context fields, not an edit of
// the title (same rule as UpdateSubtitleTracksIfMissing).

// Implements: disc-2026-10-production-countries-never-filled AC #1–#3

// UpdateProductionCountries stores a matched movie's countries. NULL keeps
// whatever is stored.
func (r *MovieRepository) UpdateProductionCountries(ctx context.Context, id string, countriesJSON models.NullString) error {
	return updateProductionCountriesColumn(ctx, r.db, "movies", id, countriesJSON)
}

// UpdateProductionCountries is the series counterpart.
func (r *SeriesRepository) UpdateProductionCountries(ctx context.Context, id string, countriesJSON models.NullString) error {
	return updateProductionCountriesColumn(ctx, r.db, "series", id, countriesJSON)
}

func updateProductionCountriesColumn(ctx context.Context, db *sql.DB, table, id string, countriesJSON models.NullString) error {
	if id == "" {
		return fmt.Errorf("%s id cannot be empty", table)
	}
	result, err := db.ExecContext(ctx,
		`UPDATE `+table+` SET production_countries = COALESCE(?, production_countries) WHERE id = ?`,
		countriesJSON, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update %s production countries: %w", table, err)
	}
	return requireOneRow(result, table, id)
}

// FillContextFacets fills genres and/or production_countries ONLY where the
// row has none. "None" is NULL or the empty string for countries, and `[]` or
// the empty string for genres (the column is NOT NULL DEFAULT '[]'); genres
// are also only filled on rows that are not user-owned (genresFillable). An
// empty argument fills nothing. written=false means there was nothing left to
// fill (another writer got there first) or the row is gone — not an error.
func (r *MovieRepository) FillContextFacets(ctx context.Context, id, genresJSON string, countriesJSON models.NullString) (bool, error) {
	return fillContextFacets(ctx, r.db, "movies", id, genresJSON, countriesJSON)
}

// FillContextFacets is the series counterpart.
func (r *SeriesRepository) FillContextFacets(ctx context.Context, id, genresJSON string, countriesJSON models.NullString) (bool, error) {
	return fillContextFacets(ctx, r.db, "series", id, genresJSON, countriesJSON)
}

func fillContextFacets(ctx context.Context, db *sql.DB, table, id, genresJSON string, countriesJSON models.NullString) (bool, error) {
	if id == "" {
		return false, fmt.Errorf("%s id cannot be empty", table)
	}
	genres := models.NullString{}
	if genresJSON != "" && genresJSON != "[]" {
		// Same Taiwan wording as every other genre write (models.GenresJSON),
		// whoever built the JSON (disc-2026-10-genre-names-simplified CR).
		var names []string
		if err := json.Unmarshal([]byte(genresJSON), &names); err != nil {
			return false, fmt.Errorf("%s genres are not a JSON list: %w", table, err)
		}
		holder := models.Movie{Genres: names}
		canonical, err := holder.GenresJSON()
		if err != nil {
			return false, fmt.Errorf("failed to encode %s genres: %w", table, err)
		}
		if canonical != "[]" {
			genres = models.NewNullString(canonical)
		}
	}
	if !genres.Valid && (!countriesJSON.Valid || countriesJSON.String == "") {
		return false, nil
	}
	result, err := db.ExecContext(ctx,
		`UPDATE `+table+` SET
			genres = CASE WHEN ? IS NOT NULL AND `+genresFillable+` THEN ? ELSE genres END,
			production_countries = CASE
				WHEN (production_countries IS NULL OR production_countries = '') THEN ?
				ELSE production_countries END
		WHERE id = ?
		  AND ((? IS NOT NULL AND `+genresFillable+`)
		    OR (? IS NOT NULL AND (production_countries IS NULL OR production_countries = '')))`,
		genres, genres, countriesJSON, id, genres, countriesJSON,
	)
	if err != nil {
		return false, fmt.Errorf("failed to fill %s context facets: %w", table, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}
	return n == 1, nil
}

// genresFillable: genres are empty AND the row is not user-owned. An empty
// list on a manual row may be the user's choice in the Metadata Editor (CR
// M1); an empty list there and "never fetched" look the same, so the user's
// rows are left alone. Countries have no editor and are filled regardless.
const genresFillable = `genres IN ('[]', '') AND COALESCE(metadata_source, '') != 'manual'`

// contextFacetsMissing is the shared filter: matched, not removed, and either
// facet fillable.
const contextFacetsMissing = `tmdb_id IS NOT NULL AND tmdb_id > 0
		  AND is_removed = 0
		  AND (production_countries IS NULL OR production_countries = ''
		    OR (` + genresFillable + `))`

// FindMissingContextFacets lists matched movies with no countries or no
// genres. Paged by id (afterID "" = from the start): a title TMDb genuinely
// has no countries for stays listed every pass, and a fixed first page would
// let such rows shut the rest out.
func (r *MovieRepository) FindMissingContextFacets(ctx context.Context, afterID string, limit int) ([]models.Movie, error) {
	query := fmt.Sprintf(`SELECT %s FROM movies WHERE %s AND id > ? ORDER BY id ASC LIMIT ?`,
		movieSelectColumns, contextFacetsMissing)
	rows, err := r.db.QueryContext(ctx, query, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query movies missing context facets: %w", err)
	}
	defer rows.Close()
	var movies []models.Movie
	for rows.Next() {
		movie, err := scanMovie(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan movie: %w", err)
		}
		movies = append(movies, movie)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating movies missing context facets: %w", err)
	}
	return movies, nil
}

// FindMissingContextFacets is the series counterpart.
func (r *SeriesRepository) FindMissingContextFacets(ctx context.Context, afterID string, limit int) ([]models.Series, error) {
	query := fmt.Sprintf(`SELECT %s FROM series WHERE %s AND id > ? ORDER BY id ASC LIMIT ?`,
		seriesSelectColumns, contextFacetsMissing)
	rows, err := r.db.QueryContext(ctx, query, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query series missing context facets: %w", err)
	}
	defer rows.Close()
	var list []models.Series
	for rows.Next() {
		s, err := scanSeries(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan series: %w", err)
		}
		list = append(list, s)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating series missing context facets: %w", err)
	}
	return list, nil
}
