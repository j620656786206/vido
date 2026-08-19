// Package services — partial-request selection (Story 13-2a, G-2/P3-002).
//
// The canonical selection shape carries [@contract-v1] (13-2a AC #1):
// `seasons` = fully-selected season numbers (sorted, deduped);
// `episodes` = partial seasons only, season-number key → sorted episode
// numbers. A season never appears on both sides, and an empty selection is
// the whole-title request (both requests columns stay NULL — byte-identical
// to every pre-13-2a row).
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

// ErrRequestInvalidSelection is the errors.Is sentinel behind the Rule 7
// REQUEST_INVALID_SELECTION wire code (13-2a AC #2/#3): a season/episode
// selection that is malformed, unknown to TMDB, or overlaps already-owned
// episodes.
var ErrRequestInvalidSelection = errors.New("invalid season/episode selection")

// InvalidSelectionError carries the zh-TW reason SEPARATELY from the English
// diagnostic chain, so the handler renders a clean Rule 3 message without the
// sentinel text leaking into it. errors.Is(err, ErrRequestInvalidSelection)
// still works via Is.
type InvalidSelectionError struct{ Reason string }

func (e *InvalidSelectionError) Error() string {
	return "invalid season/episode selection: " + e.Reason
}

// Is makes the struct answer the sentinel for errors.Is chains.
func (e *InvalidSelectionError) Is(target error) bool { return target == ErrRequestInvalidSelection }

// invalidSelection builds the typed rejection with a zh-TW reason.
func invalidSelection(format string, args ...any) error {
	return &InvalidSelectionError{Reason: fmt.Sprintf(format, args...)}
}

// RequestSelection is the canonical in-memory form of a partial request.
// nil *RequestSelection = whole-title request.
type RequestSelection struct {
	// Seasons are fully-selected season numbers, sorted ascending.
	Seasons []int
	// Episodes maps a partially-selected season to its sorted episode numbers.
	Episodes map[int][]int
}

// Selection size ceilings (CR 13-2a M1). Every selected season costs one TMDB
// season-details fetch on create AND one Sonarr episode-list fetch on
// fulfilment, so an unbounded selection turns one HTTP request into dozens of
// upstream calls. The ceilings sit far above any real show (TMDB's longest
// listed series are ~90 seasons of daily programming; a season rarely exceeds
// a few hundred episodes) — they exist to bound amplification, not to shape
// product behavior.
const (
	maxSelectedSeasons  = 100
	maxSelectedEpisodes = 2000
)

// canonicalizeSelection normalizes the DTO selection (13-2a AC #1): sort +
// dedupe both sides, reject a season present on both sides, reject any
// selection on a movie. Returns nil for an empty selection (= whole title).
func canonicalizeSelection(mediaType string, seasons []int, episodes map[string][]int) (*RequestSelection, error) {
	if len(seasons) == 0 && len(episodes) == 0 {
		return nil, nil
	}
	if mediaType != models.RequestMediaTypeTV {
		return nil, invalidSelection("只有影集可以指定季或集")
	}
	if len(seasons)+len(episodes) > maxSelectedSeasons {
		return nil, invalidSelection("一次最多只能選取 %d 季", maxSelectedSeasons)
	}

	sel := &RequestSelection{Seasons: sortedUniqueInts(seasons)}
	for _, n := range sel.Seasons {
		if n < 0 {
			return nil, invalidSelection("季數不可為負（%d）", n)
		}
	}

	if len(episodes) > 0 {
		sel.Episodes = make(map[int][]int, len(episodes))
		wholeSeasons := intSet(sel.Seasons)
		totalEpisodes := 0
		for key, eps := range episodes {
			season, err := strconv.Atoi(key)
			if err != nil || season < 0 {
				return nil, invalidSelection("episodes 的季鍵必須是非負整數（%q）", key)
			}
			if _, dup := wholeSeasons[season]; dup {
				return nil, invalidSelection("第 %d 季不可同時整季選取又列出個別集", season)
			}
			if _, dup := sel.Episodes[season]; dup {
				return nil, invalidSelection("episodes 重複的季鍵（%d）", season)
			}
			cleaned := sortedUniqueInts(eps)
			if len(cleaned) == 0 {
				return nil, invalidSelection("第 %d 季未列出任何集數", season)
			}
			for _, e := range cleaned {
				if e <= 0 {
					return nil, invalidSelection("集數必須是正整數（第 %d 季：%d）", season, e)
				}
			}
			totalEpisodes += len(cleaned)
			if totalEpisodes > maxSelectedEpisodes {
				return nil, invalidSelection("一次最多只能選取 %d 集", maxSelectedEpisodes)
			}
			sel.Episodes[season] = cleaned
		}
	}
	return sel, nil
}

// SelectionFullyOwned answers whether a stored request selection is entirely
// present in the local library (CR 13-2a H1 — the 13-3a poller's completion
// rule is title-level and would otherwise complete a partial request the
// moment ANY episode of the show exists locally). A whole-title request has
// no selection and is left to the title-level rule.
func (s *RequestService) SelectionFullyOwned(ctx context.Context, tmdbID int64, seasons, episodes models.NullString) (bool, error) {
	sel, err := parseSelectionColumns(seasons, episodes)
	if err != nil {
		return false, err
	}
	if sel == nil {
		return true, nil // whole-title: the title-level rule already decided
	}

	details, err := s.tmdb.GetTVShowDetails(ctx, int(tmdbID))
	if err != nil {
		return false, fmt.Errorf("resolve tmdb target: %w", err)
	}
	owned, err := s.ownedEpisodeNumbers(ctx, tmdbID)
	if err != nil {
		return false, err
	}
	// checkSelectionOwnership answers ALREADY-IN-LIBRARY exactly when every
	// selected unit is owned — the same predicate this needs, so the two
	// consumers cannot drift.
	return errors.Is(checkSelectionOwnership(sel, owned, details), ErrRequestAlreadyInLibrary), nil
}

// validateAgainstTMDB checks every selected season/episode against the Epic-2
// TMDB chain (13-2a AC #2): season numbers must exist on the show, and listed
// episode numbers must exist in that season's episode list. Season details are
// fetched ONLY for seasons with individual episodes listed. Season 0
// (specials) follows the TMDB data — if listed, it is selectable.
func (s *RequestService) validateAgainstTMDB(ctx context.Context, tmdbID int64, sel *RequestSelection, details *tmdb.TVShowDetails) error {
	known := make(map[int]bool, len(details.Seasons))
	for _, season := range details.Seasons {
		known[season.SeasonNumber] = true
	}
	for _, n := range sel.Seasons {
		if !known[n] {
			return invalidSelection("此影集沒有第 %d 季", n)
		}
	}
	for season, eps := range sel.Episodes {
		if !known[season] {
			return invalidSelection("此影集沒有第 %d 季", season)
		}
		seasonDetails, err := s.tmdb.GetSeasonDetails(ctx, int(tmdbID), season)
		if err != nil {
			// Typed TMDB_* errors pass through untouched (resolveTitle parity).
			return err
		}
		valid := make(map[int]bool, len(seasonDetails.Episodes))
		for _, ep := range seasonDetails.Episodes {
			valid[ep.EpisodeNumber] = true
		}
		for _, e := range eps {
			if !valid[e] {
				return invalidSelection("第 %d 季沒有第 %d 集", season, e)
			}
		}
	}
	return nil
}

// checkSelectionOwnership is the episode-level owned guard (13-2a AC #3,
// ⚖️ A ruling): with a local series present,
//   - selection entirely owned  → ErrRequestAlreadyInLibrary (409)
//   - selection partially owned → ErrRequestInvalidSelection (400 — honest
//     rejection, never a silent trim; the FE tree disables owned rows so the
//     normal flow never sends an overlap)
//   - no overlap                → allowed (this is the entire point of 13-2a:
//     a partially-owned show can request what it is missing)
//
// A fully-selected season counts as owned only when every TMDB episode of
// that season is locally present (episode_count from the show's season
// summary); it overlaps when ANY of its episodes is owned.
func checkSelectionOwnership(sel *RequestSelection, owned map[int][]int, details *tmdb.TVShowDetails) error {
	if len(owned) == 0 {
		return nil
	}
	episodeCount := make(map[int]int, len(details.Seasons))
	for _, season := range details.Seasons {
		episodeCount[season.SeasonNumber] = season.EpisodeCount
	}
	ownedSet := make(map[int]map[int]bool, len(owned))
	for season, eps := range owned {
		set := make(map[int]bool, len(eps))
		for _, e := range eps {
			set[e] = true
		}
		ownedSet[season] = set
	}

	units, ownedUnits, overlapped := 0, 0, false
	for _, season := range sel.Seasons {
		units++
		ownedInSeason := len(ownedSet[season])
		if ownedInSeason > 0 {
			overlapped = true
		}
		if count := episodeCount[season]; count > 0 && ownedInSeason >= count {
			ownedUnits++
		}
	}
	for season, eps := range sel.Episodes {
		for _, e := range eps {
			units++
			if ownedSet[season][e] {
				overlapped = true
				ownedUnits++
			}
		}
	}

	switch {
	case units > 0 && ownedUnits == units:
		return fmt.Errorf("選取的內容全部已在媒體庫中: %w", ErrRequestAlreadyInLibrary)
	case overlapped:
		return invalidSelection("選取內容包含已入庫的集數，請取消勾選後重試")
	default:
		return nil
	}
}

// ownedEpisodeNumbers builds the season → owned episode numbers map for a
// locally-present series (shared by the AC #3 guard and the AC #5 coverage
// endpoint — one query path, two consumers).
func (s *RequestService) ownedEpisodeNumbers(ctx context.Context, tmdbID int64) (map[int][]int, error) {
	series, err := s.seriesRepo.FindByTMDbID(ctx, tmdbID)
	if err != nil {
		// The caller has already established ownership via FindOwnedTMDbIDs
		// (typed absence); an error here is a genuine read failure.
		return nil, fmt.Errorf("load series by tmdb_id %d: %w", tmdbID, err)
	}
	if series == nil {
		// A nil-without-error is legal on the interface even though the
		// shipped repository returns a typed not-found; wrapping a nil error
		// would print "%!w(<nil>)" (CR 13-2a L1).
		return nil, fmt.Errorf("load series by tmdb_id %d: no row returned", tmdbID)
	}
	episodes, err := s.episodeRepo.FindBySeriesID(ctx, series.ID)
	if err != nil {
		return nil, fmt.Errorf("load episodes for series %s: %w", series.ID, err)
	}
	owned := make(map[int][]int)
	for _, ep := range episodes {
		owned[ep.SeasonNumber] = append(owned[ep.SeasonNumber], ep.EpisodeNumber)
	}
	for season := range owned {
		owned[season] = sortedUniqueInts(owned[season])
	}
	return owned, nil
}

// ─── Canonical JSON (requests.seasons / requests.episodes columns) ─────────

// selectionColumns serializes the canonical selection into the two TEXT
// columns. Whole-title (nil) → both NULL, matching every pre-13-2a row.
func selectionColumns(sel *RequestSelection) (seasons, episodes models.NullString, err error) {
	if sel == nil {
		return models.NullString{}, models.NullString{}, nil
	}
	if len(sel.Seasons) > 0 {
		raw, merr := json.Marshal(sel.Seasons)
		if merr != nil {
			return seasons, episodes, fmt.Errorf("marshal seasons: %w", merr)
		}
		seasons = models.NewNullString(string(raw))
	}
	if len(sel.Episodes) > 0 {
		keyed := make(map[string][]int, len(sel.Episodes))
		for season, eps := range sel.Episodes {
			keyed[strconv.Itoa(season)] = eps
		}
		// encoding/json sorts map keys — the stored form is deterministic.
		raw, merr := json.Marshal(keyed)
		if merr != nil {
			return seasons, episodes, fmt.Errorf("marshal episodes: %w", merr)
		}
		episodes = models.NewNullString(string(raw))
	}
	return seasons, episodes, nil
}

// parseSelectionColumns rebuilds the canonical selection from a stored row
// (fulfilment + coverage read path). NULL/NULL → nil (whole title). A
// malformed stored value is a genuine data error, not fail-soft territory —
// acting on a half-parsed selection could fetch the wrong episodes.
func parseSelectionColumns(seasons, episodes models.NullString) (*RequestSelection, error) {
	if !seasons.Valid && !episodes.Valid {
		return nil, nil
	}
	sel := &RequestSelection{}
	if seasons.Valid && seasons.String != "" {
		if err := json.Unmarshal([]byte(seasons.String), &sel.Seasons); err != nil {
			return nil, fmt.Errorf("parse requests.seasons %q: %w", seasons.String, err)
		}
	}
	if episodes.Valid && episodes.String != "" {
		keyed := map[string][]int{}
		if err := json.Unmarshal([]byte(episodes.String), &keyed); err != nil {
			return nil, fmt.Errorf("parse requests.episodes %q: %w", episodes.String, err)
		}
		sel.Episodes = make(map[int][]int, len(keyed))
		for key, eps := range keyed {
			season, err := strconv.Atoi(key)
			if err != nil {
				return nil, fmt.Errorf("parse requests.episodes season key %q: %w", key, err)
			}
			sel.Episodes[season] = eps
		}
	}
	if len(sel.Seasons) == 0 && len(sel.Episodes) == 0 {
		return nil, nil
	}
	return sel, nil
}

// ─── Coverage (AC #5) ──────────────────────────────────────────────────────

// RequestCoverage is the GET /api/v1/requests/tv/:tmdb_id/coverage response
// ([@contract-v1], 13-2a AC #5 — consumer 13-2b). requested_* mirror the
// create-body vocabulary exactly, so the FE speaks ONE selection shape on
// both wire directions.
type RequestCoverage struct {
	Owned                map[string][]int `json:"owned"`
	RequestedSeasons     []int            `json:"requested_seasons"`
	RequestedEpisodes    map[string][]int `json:"requested_episodes"`
	WholeSeriesRequested bool             `json:"whole_series_requested"`
	ActiveRequest        bool             `json:"active_request"`
}

// ─── Small helpers ─────────────────────────────────────────────────────────

func sortedUniqueInts(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int]bool, len(values))
	out := make([]int, 0, len(values))
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Ints(out)
	return out
}

func intSet(values []int) map[int]struct{} {
	set := make(map[int]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return set
}
