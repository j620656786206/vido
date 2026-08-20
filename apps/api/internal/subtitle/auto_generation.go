package subtitle

import (
	"context"
	"log/slog"

	"github.com/vido/api/internal/models"
)

// AutoGenerationMaxPerRun bounds one auto-trigger round.
//
// Deliberately a package constant rather than a per-library setting, the
// PipelineConcurrencyM1 precedent: the bound exists to stop a first scan of a
// large library from spending an evening of NAS CPU in one burst, and an
// operator-tunable knob would let someone remove the very bound it exists to
// provide. Items beyond the cap are not lost — the next scan re-enumerates
// them, and the P5 pre-flight makes re-enumeration cheap for anything already
// finished.
const AutoGenerationMaxPerRun = 20

// autoEligibleStatuses is the status set the trigger will act on.
//
// Everything else is deliberately excluded: `found` is done, `skipped` and
// `no_text_source` are terminal verdicts the pipeline already reached, and the
// in-flight states (`searching`/`probing`/`extracting`/`translating`) belong to
// a run that is either still going or died mid-way — re-entering those from a
// background trigger would race a live run.
var autoEligibleStatuses = map[models.SubtitleStatus]struct{}{
	models.SubtitleStatusNotSearched:  {},
	models.SubtitleStatusNotFound:     {},
	models.SubtitleStatusUntranslated: {},
}

// AutoLibraryPolicy answers "which libraries opted in to free auto-generation".
//
// A SET rather than a per-id predicate (Rule 11, narrow but not chatty): the
// trigger needs the whole answer once per round, and asking per item would
// issue one query per candidate on a NAS-grade SQLite file.
type AutoLibraryPolicy interface {
	AutoSubtitleLibraryIDs(ctx context.Context) (map[string]struct{}, error)
}

// AutoSeriesLibraryResolver resolves an episode's owning library.
//
// It exists because `episodes` has NO library_id column — only `movies` and
// `series` carry one. An episode's opt-in therefore has to be read through its
// parent series, and an episode whose series cannot be resolved is SKIPPED
// rather than guessed: opting in is a spending-adjacent decision, and inferring
// consent from a missing row is exactly the class of assumption the 2026-08-07
// incident was made of.
type AutoSeriesLibraryResolver interface {
	FindByID(ctx context.Context, id string) (*models.Series, error)
}

// AutoGenerator runs the free lane of the pipeline over items that arrived
// missing zh-Hant subtitles, for libraries whose owner opted in.
//
// WHAT IT IS NOT: this is not the library-wide sweep that caused the 2026-08-07
// incident. Every item is processed with ProcessItemOptions.FreeOnly, so the
// two paid routes stop at the threshold and are left for the consent screen.
// `internal/cost_consent_test.go` still holds — WorkerPool.EnqueueMissing has
// no production caller, and this type does not call it.
type AutoGenerator struct {
	item      ItemProcessor
	policy    AutoLibraryPolicy
	movies    MovieGenerationFinder
	episodes  EpisodeGenerationFinder
	series    AutoSeriesLibraryResolver
	logger    *slog.Logger
	maxPerRun int
}

// AutoGeneratorOption injects one optional port.
type AutoGeneratorOption func(*AutoGenerator)

// WithAutoCandidateFinders supplies the enumeration ports. Both are optional:
// a nil finder contributes no candidates rather than failing the round.
func WithAutoCandidateFinders(movies MovieGenerationFinder, episodes EpisodeGenerationFinder) AutoGeneratorOption {
	return func(g *AutoGenerator) { g.movies, g.episodes = movies, episodes }
}

// WithAutoSeriesResolver supplies the episode→library bridge. Without it,
// episodes are skipped entirely (their opt-in cannot be established).
func WithAutoSeriesResolver(series AutoSeriesLibraryResolver) AutoGeneratorOption {
	return func(g *AutoGenerator) { g.series = series }
}

// WithAutoMaxPerRun overrides the per-round cap. Tests use it; production does
// not.
func WithAutoMaxPerRun(n int) AutoGeneratorOption {
	return func(g *AutoGenerator) {
		if n > 0 {
			g.maxPerRun = n
		}
	}
}

// NewAutoGenerator builds the trigger. A nil logger falls back to the default.
func NewAutoGenerator(item ItemProcessor, policy AutoLibraryPolicy, logger *slog.Logger, opts ...AutoGeneratorOption) *AutoGenerator {
	if logger == nil {
		logger = slog.Default()
	}
	g := &AutoGenerator{item: item, policy: policy, logger: logger, maxPerRun: AutoGenerationMaxPerRun}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// ScanCallback adapts Run for the scan-complete hook: it returns immediately
// and does the work on its own goroutine, the postScanEnrichment shape
// (cmd/api/main.go). The scanner's completion path must not wait on minutes of
// ffmpeg.
func (g *AutoGenerator) ScanCallback() func() {
	return func() {
		go g.Run(context.Background())
	}
}

// Run executes one round: read the policy, enumerate, filter, process.
//
// Synchronous on purpose — ScanCallback owns the goroutine, so tests can assert
// the outcome without synchronising on one.
func (g *AutoGenerator) Run(ctx context.Context) {
	if g.item == nil || g.policy == nil {
		return
	}

	// 9R-10a CR M1: a failed read is NOT an empty answer. On the NAS this is a
	// locked SQLite file, and treating that as "nobody opted in" would make the
	// feature silently stop working with nothing in the log to show for it.
	enabled, err := g.policy.AutoSubtitleLibraryIDs(ctx)
	if err != nil {
		g.logger.Error("subtitle auto-generation: cannot read library policy — round aborted", "error", err)
		return
	}
	if len(enabled) == 0 {
		// Routine, not exceptional: the feature ships OFF for every library.
		g.logger.Debug("subtitle auto-generation: no library opted in")
		return
	}

	refs, err := g.collect(ctx, enabled)
	if err != nil {
		g.logger.Error("subtitle auto-generation: enumeration failed — round aborted", "error", err)
		return
	}
	if len(refs) == 0 {
		g.logger.Debug("subtitle auto-generation: nothing eligible", "libraries", len(enabled))
		return
	}

	g.logger.Info("subtitle auto-generation started",
		"libraries", len(enabled), "considered", len(refs), "max_per_run", g.maxPerRun)

	var processed, deferredPaid, failed int
	for _, ref := range refs {
		outcome, err := g.item.ProcessItem(ctx, ref, ProcessItemOptions{FreeOnly: true})
		switch {
		case err != nil:
			// Per-item failure is already recorded on the item's run row by the
			// pipeline; stopping the round here would strand every later item
			// behind one unreadable file.
			failed++
			g.logger.Warn("subtitle auto-generation: item failed",
				"media_id", ref.ID, "media_type", ref.MediaType, "error", err)
		case outcome != nil && (outcome.Kind == RouteTranslate || outcome.Kind == RouteNoTextSource):
			deferredPaid++
		default:
			processed++
		}
	}

	g.logger.Info("subtitle auto-generation finished",
		"libraries", len(enabled), "considered", len(refs),
		"processed", processed, "deferred_paid", deferredPaid, "failed", failed)
}

// collect enumerates both media families and applies the three filters:
// library opt-in, eligible status, and the per-round cap. The cap spans both
// families — episodes do not get a fresh budget after movies have spent it.
func (g *AutoGenerator) collect(ctx context.Context, enabled map[string]struct{}) ([]MediaRef, error) {
	refs := make([]MediaRef, 0, g.maxPerRun)

	if g.movies != nil {
		movies, err := g.movies.FindMissingZhHantSubtitle(ctx)
		if err != nil {
			return nil, err
		}
		for _, m := range movies {
			if len(refs) >= g.maxPerRun {
				return refs, nil
			}
			if !autoEligible(m.SubtitleStatus) || !inEnabledLibrary(m.LibraryID, enabled) {
				continue
			}
			refs = append(refs, MediaRef{ID: m.ID, MediaType: models.SubtitleRunMediaMovie})
		}
	}

	if g.episodes != nil && g.series != nil {
		episodes, err := g.episodes.FindMissingZhHantSubtitle(ctx)
		if err != nil {
			return nil, err
		}
		// One lookup per distinct series, not per episode: a season of 24
		// episodes shares one parent row.
		libraryOf := map[string]models.NullString{}
		for _, e := range episodes {
			if len(refs) >= g.maxPerRun {
				return refs, nil
			}
			if !autoEligible(e.SubtitleStatus) {
				continue
			}
			libraryID, ok := libraryOf[e.SeriesID]
			if !ok {
				series, err := g.series.FindByID(ctx, e.SeriesID)
				if err != nil {
					return nil, err
				}
				if series != nil {
					libraryID = series.LibraryID
				}
				libraryOf[e.SeriesID] = libraryID
			}
			if !inEnabledLibrary(libraryID, enabled) {
				continue
			}
			refs = append(refs, MediaRef{ID: e.ID, MediaType: models.SubtitleRunMediaEpisode})
		}
	}

	return refs, nil
}

func autoEligible(status models.SubtitleStatus) bool {
	_, ok := autoEligibleStatuses[status]
	return ok
}

// inEnabledLibrary is false for a NULL library_id: an item that belongs to no
// library has no owner who could have opted it in.
func inEnabledLibrary(libraryID models.NullString, enabled map[string]struct{}) bool {
	if !libraryID.Valid || libraryID.String == "" {
		return false
	}
	_, ok := enabled[libraryID.String]
	return ok
}
