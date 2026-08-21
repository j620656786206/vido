package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// NFOLocalizerService produces an additive zh-TW `.nfo` for a movie (Story 9R-13,
// SPIKE S1). It localizes plot/title/genres/cast-roles via the shared LLM +
// glossary infra (9R-7) and writes the result to a recognized `.nfo` slot that
// players scrape WITHOUT overwriting the original (S1 free-slot strategy).
//
// Movies-first (S1 re-spec): TV `.nfo` names (tvshow.nfo / <episode>.nfo) are
// single-slot with no additive alternative — TV localization is a follow-up.
type NFOLocalizerService struct {
	translation  *TranslationService
	glossaryRepo repository.GlossaryRepositoryInterface
	logger       *slog.Logger

	// episodes enumerates a series' episodes for the 9R-13a whole-show batch.
	// Optional / nil-safe — unwired means `include_episodes` localizes the show
	// file only.
	episodes NFOEpisodeLister
}

// NewNFOLocalizerService creates the localizer. Returns nil only when no
// translation service exists at all. It deliberately does NOT check
// IsConfigured() here: since sub-2-1a keys hot-reload, configuration is a
// per-call question (IsAvailable answers it) — a boot-time check would freeze
// a keyless boot into a permanently-404 route that a later key save could
// never revive (CR sub-2-1a M1).
func NewNFOLocalizerService(translation *TranslationService, glossaryRepo repository.GlossaryRepositoryInterface, logger *slog.Logger) *NFOLocalizerService {
	if translation == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &NFOLocalizerService{
		translation:  translation,
		glossaryRepo: glossaryRepo,
		logger:       logger.With("service", "nfo_localizer"),
	}
}

// IsAvailable reports whether localization can run.
func (s *NFOLocalizerService) IsAvailable() bool {
	return s != nil && s.translation != nil && s.translation.IsConfigured()
}

// NFOLocalizeResult reports what was written.
type NFOLocalizeResult struct {
	Path       string `json:"path"`        // the zh-TW .nfo written
	BackupPath string `json:"backup_path"` // non-empty only when replace-mode backed up an original
	Replaced   bool   `json:"replaced"`    // true when both slots were occupied → backup-and-replace
}

// LocalizeMovieNFO localizes a movie's metadata to zh-TW and writes an additive
// zh-TW `.nfo`. The original `.nfo` (if any) is never overwritten in place: when
// a recognized slot is free the write is purely additive; only when BOTH slots
// are occupied does it back up the original to `.nfo.orig` and replace (S1).
func (s *NFOLocalizerService) LocalizeMovieNFO(ctx context.Context, movie models.Movie) (*NFOLocalizeResult, error) {
	if !s.IsAvailable() {
		return nil, fmt.Errorf("nfo localizer not available")
	}
	if !movie.FilePath.Valid || movie.FilePath.String == "" {
		return nil, fmt.Errorf("movie has no file path")
	}

	nfo := movieToNFO(movie)

	// Translate the localizable fields as one glossary-aware batch (9R-7).
	glossary := s.loadGlossary(ctx, movie.ID)
	localized, err := s.translateFields(ctx, nfo, glossary)
	if err != nil {
		return nil, fmt.Errorf("localize fields: %w", err)
	}

	data := marshalNFO(localized)
	return writeAdditiveNFO(movie.FilePath.String, data)
}

// loadGlossary loads the per-show glossary as translation pairs (fail-soft).
func (s *NFOLocalizerService) loadGlossary(ctx context.Context, mediaID string) []GlossaryPair {
	if s.glossaryRepo == nil {
		return nil
	}
	m, err := s.glossaryRepo.LookupByMedia(ctx, mediaID, false)
	if err != nil || len(m) == 0 {
		return nil
	}
	pairs := make([]GlossaryPair, 0, len(m))
	for src, zh := range m {
		pairs = append(pairs, GlossaryPair{Source: src, Target: zh})
	}
	return pairs
}

// translateFields translates title/plot/genres/cast-roles on a copy of nfo,
// preserving originaltitle, person names, year, rating, and uniqueids. Fail-soft
// per field (a missing translation keeps the original value).
func (s *NFOLocalizerService) translateFields(ctx context.Context, nfo MovieNFO, glossary []GlossaryPair) (MovieNFO, error) {
	// Field order is fixed so we can map results back deterministically.
	var fields []TranslationField
	fields = append(fields, TranslationField{Key: "title", Text: nfo.Title})
	fields = append(fields, TranslationField{Key: "plot", Text: nfo.Plot})
	for i, g := range nfo.Genres {
		fields = append(fields, TranslationField{Key: "genre:" + strconv.Itoa(i), Text: g})
	}
	for i, a := range nfo.Actors {
		fields = append(fields, TranslationField{Key: "role:" + strconv.Itoa(i), Text: a.Role})
	}

	byKey, err := s.translateKeyedFields(ctx, fields, glossary)
	if err != nil {
		return nfo, err
	}

	if v, ok := byKey["title"]; ok {
		nfo.Title = v
	}
	if v, ok := byKey["plot"]; ok {
		nfo.Plot = v
	}
	for i := range nfo.Genres {
		if v, ok := byKey["genre:"+strconv.Itoa(i)]; ok {
			nfo.Genres[i] = v
		}
	}
	for i := range nfo.Actors {
		if v, ok := byKey["role:"+strconv.Itoa(i)]; ok {
			nfo.Actors[i].Role = v
		}
	}
	return nfo, nil
}

// movieToNFO builds a MovieNFO from the DB record (canonical metadata). Mirrors
// NFOGenerator.GenerateMovieNFO field mapping; kept here so localization sources
// from the DB (the reader's NFOData drops genres/cast).
func movieToNFO(movie models.Movie) MovieNFO {
	nfo := MovieNFO{Title: movie.Title, Year: movie.ReleaseDate}
	if movie.OriginalTitle.Valid {
		nfo.OriginalTitle = movie.OriginalTitle.String
	}
	if movie.Overview.Valid {
		nfo.Plot = movie.Overview.String
	}
	if movie.Genres != nil {
		nfo.Genres = append([]string(nil), movie.Genres...)
	}
	if movie.VoteAverage.Valid {
		nfo.Rating = movie.VoteAverage.Float64
	}
	if movie.CreditsJSON.Valid {
		var credits CreditsData
		if err := json.Unmarshal([]byte(movie.CreditsJSON.String), &credits); err == nil {
			for _, p := range credits.Crew {
				if p.Job == "Director" {
					nfo.Directors = append(nfo.Directors, p.Name)
				}
			}
			for _, p := range credits.Cast {
				if len(nfo.Actors) >= 10 {
					break
				}
				nfo.Actors = append(nfo.Actors, NFOActor{Name: p.Name, Role: p.Character})
			}
		}
	}
	if movie.TMDbID.Valid {
		nfo.UniqueIDs = append(nfo.UniqueIDs, NFOUniqueID{Type: "tmdb", Value: fmt.Sprintf("%d", movie.TMDbID.Int64)})
	}
	if movie.IMDbID.Valid {
		nfo.UniqueIDs = append(nfo.UniqueIDs, NFOUniqueID{Type: "imdb", Value: movie.IMDbID.String})
	}
	return nfo
}

// writeAdditiveNFO writes zh-TW nfo bytes to a recognized slot next to the movie
// file using the S1 free-slot strategy. The two recognized movie-nfo names are
// `<basename>.nfo` and `movie.nfo`.
func writeAdditiveNFO(mediaFilePath string, data []byte) (*NFOLocalizeResult, error) {
	dir := filepath.Dir(mediaFilePath)
	base := strings.TrimSuffix(filepath.Base(mediaFilePath), filepath.Ext(mediaFilePath))
	filenameSlot := filepath.Join(dir, base+".nfo")
	movieSlot := filepath.Join(dir, "movie.nfo")

	filenameExists := fileExists(filenameSlot)
	movieExists := fileExists(movieSlot)

	switch {
	case !filenameExists && !movieExists:
		// No original — write the primary <basename>.nfo.
		if err := os.WriteFile(filenameSlot, data, 0o644); err != nil {
			return nil, fmt.Errorf("write nfo: %w", err)
		}
		return &NFOLocalizeResult{Path: filenameSlot}, nil
	case filenameExists && !movieExists:
		// Original at <basename>.nfo → additive write to the free movie.nfo
		// (Jellyfin shows zh-TW; Kodi keeps the original — non-destructive).
		if err := os.WriteFile(movieSlot, data, 0o644); err != nil {
			return nil, fmt.Errorf("write nfo: %w", err)
		}
		return &NFOLocalizeResult{Path: movieSlot}, nil
	case !filenameExists && movieExists:
		// Original at movie.nfo → additive write to the free <basename>.nfo
		// (Kodi shows zh-TW; Jellyfin keeps the original — non-destructive).
		if err := os.WriteFile(filenameSlot, data, 0o644); err != nil {
			return nil, fmt.Errorf("write nfo: %w", err)
		}
		return &NFOLocalizeResult{Path: filenameSlot}, nil
	default:
		// Both occupied → back up the original then replace <basename>.nfo (S1
		// backup-and-replace). Never lose the original.
		backup := filenameSlot + ".orig"
		if !fileExists(backup) {
			orig, err := os.ReadFile(filenameSlot)
			if err != nil {
				return nil, fmt.Errorf("read original nfo for backup: %w", err)
			}
			if err := os.WriteFile(backup, orig, 0o644); err != nil {
				return nil, fmt.Errorf("back up original nfo: %w", err)
			}
		}
		if err := os.WriteFile(filenameSlot, data, 0o644); err != nil {
			return nil, fmt.Errorf("write nfo: %w", err)
		}
		return &NFOLocalizeResult{Path: filenameSlot, BackupPath: backup, Replaced: true}, nil
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// translateKeyedFields is the shared translate step behind every *toNFO
// localizer (9R-13a). Extracted from translateFields so the TV path reuses the
// movie path's exact semantics instead of growing a second, drifting copy:
// blank fields are never sent, an all-blank set short-circuits without an API
// call, and a key missing from the response simply has no entry (the caller's
// `if v, ok :=` keeps the original — per-field fail-soft).
func (s *NFOLocalizerService) translateKeyedFields(ctx context.Context, fields []TranslationField, glossary []GlossaryPair) (map[string]string, error) {
	nonEmpty := fields[:0:0]
	for _, f := range fields {
		if strings.TrimSpace(f.Text) != "" {
			nonEmpty = append(nonEmpty, f)
		}
	}
	if len(nonEmpty) == 0 {
		return nil, nil
	}

	out, err := s.translation.TranslateRequest(ctx, TranslationRequest{Fields: nonEmpty, Glossary: glossary})
	if err != nil {
		return nil, err
	}

	byKey := make(map[string]string, len(out))
	for _, f := range out {
		byKey[f.Key] = f.Text
	}
	return byKey, nil
}

// ─── TV: tvshow.nfo and per-episode .nfo (Story 9R-13a) ─────────────────────
//
// TV names are SINGLE-SLOT: `tvshow.nfo` and `<episode-basename>.nfo` are the
// only names every player recognises, and language-suffixed variants are
// ignored by all three (spike 9R-S1). There is therefore no additive option the
// movie path enjoys — TV is backup-and-replace, which is why it is gated behind
// an explicit confirmation at the handler.

// NFOEpisodeLister enumerates a series' episodes for the whole-show batch.
// Narrow on purpose (Rule 11) — *repository.EpisodeRepository satisfies it;
// main.go injects it. Nil-safe: without it, `include_episodes` degrades to
// localizing the show file only.
type NFOEpisodeLister interface {
	FindBySeriesID(ctx context.Context, seriesID string) ([]models.Episode, error)
}

// SetEpisodeLister wires the episode enumeration behind the whole-show batch.
func (s *NFOLocalizerService) SetEpisodeLister(l NFOEpisodeLister) {
	s.episodes = l
}

// NFOSeriesLocalizeResult reports a whole-show run.
//
// Skipped counts episodes with no file path — a row TMDb knows about but the
// scanner has never seen on disk. That is not a failure: there is simply
// nowhere to put the file.
type NFOSeriesLocalizeResult struct {
	Show      *NFOLocalizeResult   `json:"show"`
	Episodes  []*NFOLocalizeResult `json:"episodes"`
	Succeeded int                  `json:"succeeded"`
	Failed    int                  `json:"failed"`
	Skipped   int                  `json:"skipped"`
}

// LocalizeTVShowNFO localizes a series' metadata to zh-TW and writes
// `tvshow.nfo` in the show folder, backing up any original first.
func (s *NFOLocalizerService) LocalizeTVShowNFO(ctx context.Context, series models.Series) (*NFOLocalizeResult, error) {
	if !s.IsAvailable() {
		return nil, fmt.Errorf("nfo localizer not available")
	}
	if !series.FilePath.Valid || series.FilePath.String == "" {
		return nil, fmt.Errorf("series has no folder path — scan the media library first")
	}
	// CR M3: the field is a free-form string, and tvshow.nfo is JOINED onto it.
	// If it ever holds a file (SaveSeriesFromTMDb takes a file path, even though
	// nothing calls it today) the write fails deep inside os.WriteFile with
	// "not a directory" against a path the operator never typed. Say what is
	// actually wrong instead.
	if info, err := os.Stat(series.FilePath.String); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("series folder %q is not a readable directory — rescan the media library", series.FilePath.String)
	}

	nfo := seriesToNFO(series)

	glossary := s.loadGlossary(ctx, series.ID)
	localized, err := s.translateSeriesFields(ctx, nfo, glossary)
	if err != nil {
		return nil, fmt.Errorf("localize fields: %w", err)
	}

	// 🔴 series.FilePath is ALREADY the show FOLDER (media_ingest_service.go
	// stores SeriesDirFor's output, which climbs past `Season 02`/`S02`/`第二季`).
	// Taking filepath.Dir of it — the movie path's shape, where FilePath is a
	// FILE — would write tvshow.nfo one level UP, into the library root: no
	// player would find it and the user's library root would be polluted.
	return writeReplaceNFO(filepath.Join(series.FilePath.String, "tvshow.nfo"), marshalNFO(localized))
}

// LocalizeEpisodeNFO localizes one episode and writes `<basename>.nfo` beside
// the episode file. showTitle populates <showtitle> and is passed in so a
// whole-show batch resolves the series exactly once.
func (s *NFOLocalizerService) LocalizeEpisodeNFO(ctx context.Context, episode models.Episode, showTitle string) (*NFOLocalizeResult, error) {
	if !s.IsAvailable() {
		return nil, fmt.Errorf("nfo localizer not available")
	}
	if !episode.FilePath.Valid || episode.FilePath.String == "" {
		return nil, fmt.Errorf("episode has no file path — scan the media library first")
	}

	// 🔴 The glossary keys on the parent SERIES, never the episode. A per-episode
	// key gives every episode of one show its own private vocabulary, so the same
	// character is rendered differently in episode 3 and episode 4 — the exact
	// bug sub-5-5 CR H1 fixed on the subtitle side.
	return s.localizeEpisodeWithGlossary(ctx, episode, showTitle, s.loadGlossary(ctx, episode.SeriesID))
}

// localizeEpisodeWithGlossary is LocalizeEpisodeNFO with the show glossary
// already in hand.
//
// CR M1: a 24-episode run used to re-read the SAME series glossary 24 times —
// an N+1 on exactly the axis AC #4.3 called out, just on the glossary rather
// than the series row. The whole-show batch now loads it once and threads it
// through; the single-episode entry point above still loads its own.
func (s *NFOLocalizerService) localizeEpisodeWithGlossary(ctx context.Context, episode models.Episode, showTitle string, glossary []GlossaryPair) (*NFOLocalizeResult, error) {
	nfo := episodeToNFO(episode, showTitle)

	localized, err := s.translateEpisodeFields(ctx, nfo, glossary)
	if err != nil {
		return nil, fmt.Errorf("localize fields: %w", err)
	}

	// An episode's FilePath IS a file, so the directory has to be derived —
	// the opposite of the series case above.
	dir := filepath.Dir(episode.FilePath.String)
	base := strings.TrimSuffix(filepath.Base(episode.FilePath.String), filepath.Ext(episode.FilePath.String))
	return writeReplaceNFO(filepath.Join(dir, base+".nfo"), marshalNFO(localized))
}

// LocalizeSeriesNFOWithEpisodes localizes tvshow.nfo and then every episode.
//
// Fail-soft per episode (Rule 13 case 3): one unreadable file must not abandon
// the other 23. The show file is NOT fail-soft — if it cannot be written the
// caller asked for something that could not start.
func (s *NFOLocalizerService) LocalizeSeriesNFOWithEpisodes(ctx context.Context, series models.Series) (*NFOSeriesLocalizeResult, error) {
	show, err := s.LocalizeTVShowNFO(ctx, series)
	if err != nil {
		return nil, err
	}

	out := &NFOSeriesLocalizeResult{Show: show, Episodes: []*NFOLocalizeResult{}}
	if s.episodes == nil {
		s.logger.Warn("no episode lister wired — localized the show file only",
			"series_id", series.ID)
		return out, nil
	}

	episodes, err := s.episodes.FindBySeriesID(ctx, series.ID)
	if err != nil {
		// The show file is already written; reporting a hard failure would
		// hide that. Degrade to show-only and say so.
		s.logger.Warn("could not enumerate episodes — localized the show file only",
			"series_id", series.ID, "error", err)
		return out, nil
	}

	// The series is resolved ONCE; showTitle rides along so no episode goes
	// back to the database for its parent (N+1).
	showTitle := series.Title
	// CR M1: one glossary read for the whole show, not one per episode.
	glossary := s.loadGlossary(ctx, series.ID)
	for _, ep := range episodes {
		if !ep.FilePath.Valid || ep.FilePath.String == "" {
			out.Skipped++
			continue
		}
		res, err := s.localizeEpisodeWithGlossary(ctx, ep, showTitle, glossary)
		if err != nil {
			out.Failed++
			s.logger.Warn("episode nfo localization failed — continuing with the rest",
				"series_id", series.ID, "episode_id", ep.ID,
				"season", ep.SeasonNumber, "episode", ep.EpisodeNumber, "error", err)
			continue
		}
		out.Succeeded++
		out.Episodes = append(out.Episodes, res)
	}

	s.logger.Info("series nfo localization complete",
		"series_id", series.ID, "succeeded", out.Succeeded,
		"failed", out.Failed, "skipped", out.Skipped)
	return out, nil
}

// translateSeriesFields mirrors translateFields for the tvshow shape.
func (s *NFOLocalizerService) translateSeriesFields(ctx context.Context, nfo SeriesNFO, glossary []GlossaryPair) (SeriesNFO, error) {
	var fields []TranslationField
	fields = append(fields, TranslationField{Key: "title", Text: nfo.Title})
	fields = append(fields, TranslationField{Key: "plot", Text: nfo.Plot})
	for i, g := range nfo.Genres {
		fields = append(fields, TranslationField{Key: "genre:" + strconv.Itoa(i), Text: g})
	}
	for i, a := range nfo.Actors {
		fields = append(fields, TranslationField{Key: "role:" + strconv.Itoa(i), Text: a.Role})
	}

	byKey, err := s.translateKeyedFields(ctx, fields, glossary)
	if err != nil {
		return nfo, err
	}

	if v, ok := byKey["title"]; ok {
		nfo.Title = v
	}
	if v, ok := byKey["plot"]; ok {
		nfo.Plot = v
	}
	for i := range nfo.Genres {
		if v, ok := byKey["genre:"+strconv.Itoa(i)]; ok {
			nfo.Genres[i] = v
		}
	}
	for i := range nfo.Actors {
		if v, ok := byKey["role:"+strconv.Itoa(i)]; ok {
			nfo.Actors[i].Role = v
		}
	}
	return nfo, nil
}

// translateEpisodeFields translates the only two localizable episode fields.
func (s *NFOLocalizerService) translateEpisodeFields(ctx context.Context, nfo EpisodeNFO, glossary []GlossaryPair) (EpisodeNFO, error) {
	fields := []TranslationField{
		{Key: "title", Text: nfo.Title},
		{Key: "plot", Text: nfo.Plot},
	}

	byKey, err := s.translateKeyedFields(ctx, fields, glossary)
	if err != nil {
		return nfo, err
	}

	if v, ok := byKey["title"]; ok {
		nfo.Title = v
	}
	if v, ok := byKey["plot"]; ok {
		nfo.Plot = v
	}
	// ShowTitle is deliberately NOT translated here: it is the parent series'
	// title, localized once by LocalizeTVShowNFO. Translating it per episode
	// would spend tokens to produce the same string N times — and risk N
	// different renderings of one show name.
	return nfo, nil
}

// seriesToNFO mirrors NFOGenerator.GenerateSeriesNFO's field mapping (kept here
// for the same reason movieToNFO is: localization sources from the DB record).
func seriesToNFO(series models.Series) SeriesNFO {
	nfo := SeriesNFO{Title: series.Title, Year: series.FirstAirDate}
	if series.OriginalTitle.Valid {
		nfo.OriginalTitle = series.OriginalTitle.String
	}
	if series.Overview.Valid {
		nfo.Plot = series.Overview.String
	}
	if series.Genres != nil {
		nfo.Genres = append([]string(nil), series.Genres...)
	}
	if series.VoteAverage.Valid {
		nfo.Rating = series.VoteAverage.Float64
	}
	if series.PosterPath.Valid {
		nfo.Thumb = series.PosterPath.String
	}
	if series.Status.Valid {
		nfo.Status = series.Status.String
	}
	if series.CreditsJSON.Valid {
		var credits CreditsData
		if err := json.Unmarshal([]byte(series.CreditsJSON.String), &credits); err == nil {
			for _, p := range credits.Cast {
				if len(nfo.Actors) >= 10 {
					break
				}
				nfo.Actors = append(nfo.Actors, NFOActor{Name: p.Name, Role: p.Character})
			}
		}
	}
	if series.TMDbID.Valid {
		nfo.UniqueIDs = append(nfo.UniqueIDs, NFOUniqueID{Type: "tmdb", Value: fmt.Sprintf("%d", series.TMDbID.Int64)})
	}
	if series.IMDbID.Valid {
		nfo.UniqueIDs = append(nfo.UniqueIDs, NFOUniqueID{Type: "imdb", Value: series.IMDbID.String})
	}
	return nfo
}

// episodeToNFO builds an EpisodeNFO from the DB record. `Overview` maps to
// <plot>: the episodes table has no plot column, the same shape movieToNFO
// uses for movies.
func episodeToNFO(episode models.Episode, showTitle string) EpisodeNFO {
	nfo := EpisodeNFO{
		ShowTitle: showTitle,
		Season:    episode.SeasonNumber,
		Episode:   episode.EpisodeNumber,
	}
	if episode.Title.Valid {
		nfo.Title = episode.Title.String
	}
	if episode.Overview.Valid {
		nfo.Plot = episode.Overview.String
	}
	if episode.AirDate.Valid {
		nfo.Aired = episode.AirDate.String
	}
	if episode.Runtime.Valid {
		nfo.Runtime = int(episode.Runtime.Int64)
	}
	if episode.VoteAverage.Valid {
		nfo.Rating = episode.VoteAverage.Float64
	}
	if episode.StillPath.Valid {
		nfo.Thumb = episode.StillPath.String
	}
	if episode.TMDbID.Valid {
		nfo.UniqueIDs = append(nfo.UniqueIDs, NFOUniqueID{Type: "tmdb", Value: fmt.Sprintf("%d", episode.TMDbID.Int64)})
	}
	return nfo
}

// writeReplaceNFO writes zh-TW nfo bytes to a SINGLE-SLOT target, backing up any
// original exactly once.
//
// Separate from writeAdditiveNFO rather than a mode flag on it: that function
// encodes the MOVIE two-slot layout (`<basename>.nfo` + `movie.nfo`), which has
// no TV counterpart. Teaching it a third layout would make the movie path's
// free-slot logic conditional on media type for no shared behaviour.
//
// The backup is written only when one does not already exist, so re-running
// localization can never overwrite the user's true original with a previously
// localized copy.
func writeReplaceNFO(targetPath string, data []byte) (*NFOLocalizeResult, error) {
	if !fileExists(targetPath) {
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			return nil, fmt.Errorf("write nfo: %w", err)
		}
		return &NFOLocalizeResult{Path: targetPath}, nil
	}

	backup := targetPath + ".orig"
	if !fileExists(backup) {
		orig, err := os.ReadFile(targetPath)
		if err != nil {
			return nil, fmt.Errorf("read original nfo for backup: %w", err)
		}
		if err := os.WriteFile(backup, orig, 0o644); err != nil {
			return nil, fmt.Errorf("back up original nfo: %w", err)
		}
	}
	if err := os.WriteFile(targetPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("write nfo: %w", err)
	}
	return &NFOLocalizeResult{Path: targetPath, BackupPath: backup, Replaced: true}, nil
}
