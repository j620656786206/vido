package tmdb

import (
	"context"
	"fmt"
	"net/url"
)

// Credits endpoints (sub-7-3). Cast/character names are what the glossary
// seeder plants into `show_glossary` at scan time, so the same title is
// fetched twice — once in the caller's source language (en-US) for the
// `term_src` side and once in zh-TW for the `term_zh` side — and paired by
// credit id. TMDb translates a PERSON's `name` when a translation exists
// (`original_name` stays put); `character` is whatever the editor typed and
// is often untranslated, which is why the seeder must check for Han
// characters instead of trusting the language parameter.
//
// NOTE: these are intentionally NOT part of ClientInterface (same reasoning as
// SearchPeople): adding them there would force every ClientInterface mock to
// grow four methods. Consumers depend on CreditsClientInterface instead —
// CacheService adds a language-keyed cache in front of *Client.

// CreditsClientInterface is the narrow surface the credits consumers depend
// on. *Client and *CacheService both satisfy it.
type CreditsClientInterface interface {
	GetMovieCreditsWithLanguage(ctx context.Context, movieID int, language string) (*MovieCredits, error)
	GetTVAggregateCreditsWithLanguage(ctx context.Context, tvID int, language string) (*TVAggregateCredits, error)
}

// CreditCast is one cast row of GET /movie/{id}/credits.
type CreditCast struct {
	ID                 int     `json:"id" example:"819"`
	Name               string  `json:"name" example:"Edward Norton"`
	OriginalName       string  `json:"original_name" example:"Edward Norton"`
	Character          string  `json:"character" example:"The Narrator"`
	CreditID           string  `json:"credit_id" example:"52fe4250c3a36847f80149f3"`
	Order              int     `json:"order" example:"0"`
	ProfilePath        *string `json:"profile_path"`
	KnownForDepartment string  `json:"known_for_department" example:"Acting"`
}

// CreditCrew is one crew row of GET /movie/{id}/credits.
type CreditCrew struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	OriginalName string  `json:"original_name"`
	Department   string  `json:"department" example:"Directing"`
	Job          string  `json:"job" example:"Director"`
	CreditID     string  `json:"credit_id"`
	ProfilePath  *string `json:"profile_path"`
}

// MovieCredits is the response of GET /movie/{id}/credits.
type MovieCredits struct {
	ID   int          `json:"id"`
	Cast []CreditCast `json:"cast"`
	Crew []CreditCrew `json:"crew"`
}

// AggregateRole is one character an actor played across a series
// (GET /tv/{id}/aggregate_credits → cast[].roles[]).
type AggregateRole struct {
	CreditID     string `json:"credit_id"`
	Character    string `json:"character" example:"Walter White"`
	EpisodeCount int    `json:"episode_count" example:"62"`
}

// AggregateCast is one actor row of GET /tv/{id}/aggregate_credits. Unlike the
// per-season /credits endpoint it lists every recurring role of the show,
// which is what a per-show glossary wants.
type AggregateCast struct {
	ID                 int             `json:"id"`
	Name               string          `json:"name"`
	OriginalName       string          `json:"original_name"`
	Roles              []AggregateRole `json:"roles"`
	TotalEpisodeCount  int             `json:"total_episode_count"`
	Order              int             `json:"order"`
	ProfilePath        *string         `json:"profile_path"`
	KnownForDepartment string          `json:"known_for_department"`
}

// AggregateJob is one job a crew member held across a series.
type AggregateJob struct {
	CreditID     string `json:"credit_id"`
	Job          string `json:"job"`
	EpisodeCount int    `json:"episode_count"`
}

// AggregateCrew is one crew row of GET /tv/{id}/aggregate_credits.
type AggregateCrew struct {
	ID                int            `json:"id"`
	Name              string         `json:"name"`
	OriginalName      string         `json:"original_name"`
	Department        string         `json:"department"`
	Jobs              []AggregateJob `json:"jobs"`
	TotalEpisodeCount int            `json:"total_episode_count"`
	ProfilePath       *string        `json:"profile_path"`
}

// TVAggregateCredits is the response of GET /tv/{id}/aggregate_credits.
type TVAggregateCredits struct {
	ID   int             `json:"id"`
	Cast []AggregateCast `json:"cast"`
	Crew []AggregateCrew `json:"crew"`
}

// GetMovieCredits retrieves a movie's cast and crew in the client's default
// language.
func (c *Client) GetMovieCredits(ctx context.Context, movieID int) (*MovieCredits, error) {
	return c.GetMovieCreditsWithLanguage(ctx, movieID, c.language)
}

// GetMovieCreditsWithLanguage retrieves a movie's cast and crew
// (GET /movie/{id}/credits) in a specific language.
func (c *Client) GetMovieCreditsWithLanguage(ctx context.Context, movieID int, language string) (*MovieCredits, error) {
	if movieID <= 0 {
		return nil, NewBadRequestError("movie ID must be greater than 0")
	}
	queryParams := url.Values{}
	if language != "" {
		queryParams.Set("language", language)
	}
	var result MovieCredits
	if err := c.Get(ctx, fmt.Sprintf("/movie/%d/credits", movieID), queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to get movie credits: %w", err)
	}
	return &result, nil
}

// GetTVAggregateCredits retrieves a series' aggregated cast and crew in the
// client's default language.
func (c *Client) GetTVAggregateCredits(ctx context.Context, tvID int) (*TVAggregateCredits, error) {
	return c.GetTVAggregateCreditsWithLanguage(ctx, tvID, c.language)
}

// GetTVAggregateCreditsWithLanguage retrieves a series' aggregated cast and
// crew (GET /tv/{id}/aggregate_credits) in a specific language.
func (c *Client) GetTVAggregateCreditsWithLanguage(ctx context.Context, tvID int, language string) (*TVAggregateCredits, error) {
	if tvID <= 0 {
		return nil, NewBadRequestError("TV show ID must be greater than 0")
	}
	queryParams := url.Values{}
	if language != "" {
		queryParams.Set("language", language)
	}
	var result TVAggregateCredits
	if err := c.Get(ctx, fmt.Sprintf("/tv/%d/aggregate_credits", tvID), queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to get tv aggregate credits: %w", err)
	}
	return &result, nil
}
