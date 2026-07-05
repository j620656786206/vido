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
}

// NewNFOLocalizerService creates the localizer. Returns nil when translation is
// unavailable (graceful degradation — the feature is simply disabled).
func NewNFOLocalizerService(translation *TranslationService, glossaryRepo repository.GlossaryRepositoryInterface, logger *slog.Logger) *NFOLocalizerService {
	if translation == nil || !translation.IsConfigured() {
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

	// Drop empty-text fields so the model isn't asked to translate blanks.
	nonEmpty := fields[:0:0]
	for _, f := range fields {
		if strings.TrimSpace(f.Text) != "" {
			nonEmpty = append(nonEmpty, f)
		}
	}
	if len(nonEmpty) == 0 {
		return nfo, nil
	}

	out, err := s.translation.TranslateRequest(ctx, TranslationRequest{Fields: nonEmpty, Glossary: glossary})
	if err != nil {
		return nfo, err
	}

	byKey := make(map[string]string, len(out))
	for _, f := range out {
		byKey[f.Key] = f.Text
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
