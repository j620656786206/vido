package handlers

import (
	"context"
	"log/slog"
	"strings"

	"github.com/vido/api/internal/models"
)

// MediaCountryResolver returns a title's production country codes. The manual
// subtitle paths need them for the own-wording rule (zhtw.KeepsOwnWording): a
// converted mainland / Hong Kong / Macau subtitle keeps its own wording.
type MediaCountryResolver func(ctx context.Context, mediaType, mediaID string) ([]string, error)

// countryMovieFinder / countrySeriesFinder are the one repository method the
// resolver needs; repos.Movies and repos.Series satisfy them.
type countryMovieFinder interface {
	FindByID(ctx context.Context, id string) (*models.Movie, error)
}

type countrySeriesFinder interface {
	FindByID(ctx context.Context, id string) (*models.Series, error)
}

// NewRepoCountryResolver reads production countries off the stored movie or
// series row. A missing row or an unwired repository yields no countries.
func NewRepoCountryResolver(movies countryMovieFinder, series countrySeriesFinder) MediaCountryResolver {
	return func(ctx context.Context, mediaType, mediaID string) ([]string, error) {
		var countries []models.ProductionCountry
		switch mediaType {
		case "movie":
			if movies == nil {
				return nil, nil
			}
			movie, err := movies.FindByID(ctx, mediaID)
			if err != nil {
				return nil, err
			}
			if movie != nil {
				countries = movie.ProductionCountries
			}
		case "series":
			if series == nil {
				return nil, nil
			}
			show, err := series.FindByID(ctx, mediaID)
			if err != nil {
				return nil, err
			}
			if show != nil {
				countries = show.ProductionCountries
			}
		}
		var codes []string
		for _, c := range countries {
			if code := strings.TrimSpace(c.ISO3166_1); code != "" {
				codes = append(codes, code)
			}
		}
		return codes, nil
	}
}

// SetCountryResolver wires the production-country lookup for the manual
// download and convert paths. Unset, no title keeps its own wording.
func (h *SubtitleHandler) SetCountryResolver(r MediaCountryResolver) {
	h.countries = r
}

// countriesFor never fails the request: a lookup error only costs the
// own-wording exemption, so the title gets the Taiwan wording and is logged.
func (h *SubtitleHandler) countriesFor(ctx context.Context, mediaType, mediaID string) []string {
	if h.countries == nil {
		return nil
	}
	codes, err := h.countries(ctx, mediaType, mediaID)
	if err != nil {
		slog.Warn("Production-country lookup failed — converting with Taiwan wording",
			"media_id", mediaID, "media_type", mediaType, "error", err)
		return nil
	}
	return codes
}
