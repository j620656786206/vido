package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// GlossaryScopeResolverInterface turns a LOCAL media id (movie / series /
// episode row id — whatever a caller happens to hold) into the glossary
// SCOPE its terms live in (sub-7-1 AC #2). Every glossary consumer resolves
// first and reads the repository by scope; the repository itself never sees a
// media id again.
type GlossaryScopeResolverInterface interface {
	Resolve(ctx context.Context, mediaID string) (string, error)
}

// The three finders the resolver needs, narrowed to FindByID so tests fake
// one method each instead of the repositories' dozens.
type glossarySeriesFinder interface {
	FindByID(ctx context.Context, id string) (*models.Series, error)
}
type glossaryMovieFinder interface {
	FindByID(ctx context.Context, id string) (*models.Movie, error)
}
type glossaryEpisodeFinder interface {
	FindByID(ctx context.Context, id string) (*models.Episode, error)
}

// glossaryScopeMover is the one repository method the resolver writes through
// (AC #3, the local→tmdb upgrade). Optional: nil = resolve only.
type glossaryScopeMover interface {
	MigrateScope(ctx context.Context, from, to string) (moved, skipped int64, err error)
}

// GlossaryScopeSeeder is the seed-on-first-resolve hook
// (backlog-glossary-seed-existing-library-and-parse-queue): called every time
// Resolve answers with a SHARED scope, after the local→tmdb move, so the
// user's own terms are in the drawer before any TMDb seed is. The
// implementation (GlossarySeeder.EnsureSeeded) makes itself idempotent and
// cheap; the resolver just guarantees the moment.
type GlossaryScopeSeeder interface {
	EnsureSeeded(ctx context.Context, scope, mediaID string)
}

// GlossaryScopeResolver resolves media ids to glossary scopes. It does NOT
// cache: a TMDb match can land minutes after the file row did (scan writes
// the row first, enrichment matches later), and a cached `local:` answer
// would keep feeding the unshared drawer after the match. One indexed
// SELECT per resolve is the price of always being right about that.
type GlossaryScopeResolver struct {
	series   glossarySeriesFinder
	movies   glossaryMovieFinder
	episodes glossaryEpisodeFinder
	mover    glossaryScopeMover
	seeder   GlossaryScopeSeeder
	logger   *slog.Logger
}

// NewGlossaryScopeResolver wires the three media finders and (optionally) the
// glossary repository that performs the local→tmdb move. Any nil finder is
// treated as "never found"; a nil mover disables the upgrade step.
func NewGlossaryScopeResolver(
	series glossarySeriesFinder,
	movies glossaryMovieFinder,
	episodes glossaryEpisodeFinder,
	mover glossaryScopeMover,
	logger *slog.Logger,
) *GlossaryScopeResolver {
	if logger == nil {
		logger = slog.Default()
	}
	return &GlossaryScopeResolver{series: series, movies: movies, episodes: episodes, mover: mover, logger: logger}
}

var _ GlossaryScopeResolverInterface = (*GlossaryScopeResolver)(nil)

// SetSeeder wires the seed-on-first-resolve hook. Optional: nil = no seeding.
func (r *GlossaryScopeResolver) SetSeeder(seeder GlossaryScopeSeeder) {
	r.seeder = seeder
}

// Resolve maps mediaID to its scope. Lookup order follows how the callers key
// their glossaries: series first (the pipeline already hands episodes in as
// their series id via ShowKey), then movie, then episode → its series. A row
// that exists but has no TMDb id resolves to `local:<its own show id>`; an id
// nothing knows resolves to `local:<mediaID>` — both keep the pre-sub-7-1
// behaviour under a new name. Repository errors other than not-found are
// returned: callers fail-soft (translate without a glossary) and log.
//
// When the answer is a `tmdb:*` scope, the terms that were written under the
// old `local:` key before the match landed are moved into it (AC #3) — every
// resolve, because there is no cache; the UPDATE is indexed and touches zero
// rows once the move has happened.
func (r *GlossaryScopeResolver) Resolve(ctx context.Context, mediaID string) (string, error) {
	id := strings.TrimSpace(mediaID)
	if id == "" {
		return "", &models.ValidationError{Field: "media_id", Message: "media_id is required"}
	}

	scope, showID, err := r.lookup(ctx, id)
	if err != nil {
		return "", err
	}
	if models.IsSharedGlossaryScope(scope) {
		r.upgradeLocal(ctx, showID, scope)
		// An episode id may also have been used as a key by a pre-ShowKey
		// caller; sweep that drawer too when it differs from the show's.
		if showID != id {
			r.upgradeLocal(ctx, id, scope)
		}
		// Seed AFTER the move: MigrateScope never overwrites, so whatever the
		// user confirmed while the show was unmatched must land first.
		if r.seeder != nil {
			r.seeder.EnsureSeeded(ctx, scope, showID)
		}
	}
	return scope, nil
}

// lookup returns the scope and the SHOW-level id the local fallback should key
// on (the series id for an episode, the id itself otherwise).
func (r *GlossaryScopeResolver) lookup(ctx context.Context, id string) (scope, showID string, err error) {
	if r.series != nil {
		s, err := r.series.FindByID(ctx, id)
		switch {
		case err == nil:
			if s.TMDbID.Valid {
				return models.GlossaryScopeTV(s.TMDbID.Int64), id, nil
			}
			return models.GlossaryScopeLocal(id), id, nil
		case !isGlossaryNotFound(err):
			return "", "", fmt.Errorf("resolve glossary scope (series %s): %w", id, err)
		}
	}
	if r.movies != nil {
		m, err := r.movies.FindByID(ctx, id)
		switch {
		case err == nil:
			if m.TMDbID.Valid {
				return models.GlossaryScopeMovie(m.TMDbID.Int64), id, nil
			}
			return models.GlossaryScopeLocal(id), id, nil
		case !isGlossaryNotFound(err):
			return "", "", fmt.Errorf("resolve glossary scope (movie %s): %w", id, err)
		}
	}
	if r.episodes != nil && r.series != nil {
		e, err := r.episodes.FindByID(ctx, id)
		switch {
		case err == nil && e.SeriesID != "":
			s, serr := r.series.FindByID(ctx, e.SeriesID)
			switch {
			case serr == nil:
				if s.TMDbID.Valid {
					return models.GlossaryScopeTV(s.TMDbID.Int64), e.SeriesID, nil
				}
				return models.GlossaryScopeLocal(e.SeriesID), e.SeriesID, nil
			case !isGlossaryNotFound(serr):
				return "", "", fmt.Errorf("resolve glossary scope (episode %s → series %s): %w", id, e.SeriesID, serr)
			}
			// Orphan episode: its series row is gone — key on the series id it
			// still remembers, so sibling episodes keep sharing a drawer.
			return models.GlossaryScopeLocal(e.SeriesID), e.SeriesID, nil
		case err != nil && !isGlossaryNotFound(err):
			return "", "", fmt.Errorf("resolve glossary scope (episode %s): %w", id, err)
		}
	}
	return models.GlossaryScopeLocal(id), id, nil
}

// upgradeLocal moves `local:<localID>` into the shared scope (AC #3). Fail-soft:
// a failed move is logged and the resolve still succeeds — the shared drawer is
// still the right answer, the stale local terms simply wait for the next call.
func (r *GlossaryScopeResolver) upgradeLocal(ctx context.Context, localID, shared string) {
	if r.mover == nil {
		return
	}
	from := models.GlossaryScopeLocal(localID)
	moved, skipped, err := r.mover.MigrateScope(ctx, from, shared)
	if err != nil {
		r.logger.Warn("glossary scope upgrade failed — local terms stay behind until the next resolve",
			"from", from, "to", shared, "error", err)
		return
	}
	if moved > 0 || skipped > 0 {
		r.logger.Info("glossary terms moved into the shared TMDb drawer",
			"from", from, "to", shared, "moved", moved, "kept_local_because_shared_has_them", skipped)
	}
}

// isGlossaryNotFound recognises the three repositories' not-found signals:
// movies/series wrap sql.ErrNoRows, episodes use their own sentinel.
func isGlossaryNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, repository.ErrEpisodeNotFound)
}

// localScopeFallback is what a consumer uses when NO resolver is wired (tests,
// partial wiring): the pre-sub-7-1 key under its new name, so behaviour is
// unchanged rather than broken.
func localScopeFallback(mediaID string) string {
	return models.GlossaryScopeLocal(mediaID)
}
