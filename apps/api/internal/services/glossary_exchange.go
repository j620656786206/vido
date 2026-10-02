package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// Story sub-8-1: share a show's glossary as a FILE — export one, hand it to a
// friend, they import it. Deliberately no network transport: whether anyone
// wants to share at all decides whether sub-8-2 (a shared wall) is worth
// building.

// GlossaryExportFormat / GlossaryExportVersion identify the file.
const (
	GlossaryExportFormat  = "vido-glossary"
	GlossaryExportVersion = 1

	// glossaryImportMaxTerms caps one file. A show's glossary is tens to a few
	// hundred rows; a file with tens of thousands is not a glossary.
	glossaryImportMaxTerms = 5000
	// glossaryImportMaxRunes caps either side of one term.
	glossaryImportMaxRunes = 200
)

// Sentinels the handler maps to GLOSSARY_* codes (Rule 7).
var (
	// ErrGlossaryNotShareable — the media resolves to a local:* scope (no TMDb
	// match), so there is no id a friend's library would share.
	ErrGlossaryNotShareable = errors.New("glossary: not shareable — this title has no TMDb match")
	// ErrGlossaryScopeMismatch — the file was exported from a different show.
	ErrGlossaryScopeMismatch = errors.New("glossary: file is for a different title")
	// ErrGlossaryImportInvalid — not a vido-glossary v1 file, or over the limits.
	ErrGlossaryImportInvalid = errors.New("glossary: not a valid vido-glossary file")
)

// GlossaryExportTerm is one row of the file.
type GlossaryExportTerm struct {
	TermSrc   string `json:"term_src"`
	TermZh    string `json:"term_zh"`
	Source    string `json:"source"`
	Confirmed bool   `json:"confirmed"`
}

// GlossaryExport is the shareable file.
//
// [@contract-v1] (sub-8-1 AC #1) — the on-disk format two installs exchange.
// Adding an optional field is additive; changing or removing one bumps
// `version`, and importers refuse versions they do not know.
type GlossaryExport struct {
	Format     string               `json:"format"`
	Version    int                  `json:"version"`
	ExportedAt time.Time            `json:"exported_at"`
	Scope      string               `json:"scope"`
	Title      string               `json:"title"`
	Language   string               `json:"language"`
	Terms      []GlossaryExportTerm `json:"terms"`
}

// GlossaryImportConflict is a term both sides have with DIFFERENT renderings.
// Nothing is overwritten on import; the user picks per conflict (keep mine /
// use theirs — the latter is an ordinary edit of row ID).
type GlossaryImportConflict struct {
	ID            string `json:"id"`
	TermSrc       string `json:"term_src"`
	Mine          string `json:"mine"`
	Theirs        string `json:"theirs"`
	MineSource    string `json:"mine_source"`
	MineConfirmed bool   `json:"mine_confirmed"`
}

// GlossaryImportResult is what an import did.
//
// [@contract-v1] (sub-8-1 AC #2).
type GlossaryImportResult struct {
	Imported  int                      `json:"imported"`
	Skipped   int                      `json:"skipped"`
	Conflicts []GlossaryImportConflict `json:"conflicts"`
	Title     string                   `json:"title"`
}

// GlossaryTitleLookup returns a media item's display title ("" when unknown).
type GlossaryTitleLookup func(ctx context.Context, mediaID string) string

// GlossaryExchangeService exports and imports glossary files.
type GlossaryExchangeService struct {
	repo   repository.GlossaryRepositoryInterface
	scopes GlossaryScopeResolverInterface
	title  GlossaryTitleLookup
	now    func() time.Time
	logger *slog.Logger
}

// NewGlossaryExchangeService wires the service. title may be nil.
func NewGlossaryExchangeService(repo repository.GlossaryRepositoryInterface, scopes GlossaryScopeResolverInterface, title GlossaryTitleLookup, logger *slog.Logger) *GlossaryExchangeService {
	if logger == nil {
		logger = slog.Default()
	}
	return &GlossaryExchangeService{repo: repo, scopes: scopes, title: title, now: time.Now, logger: logger.With("service", "glossary_exchange")}
}

func (s *GlossaryExchangeService) sharedScope(ctx context.Context, mediaID string) (string, error) {
	if strings.TrimSpace(mediaID) == "" {
		return "", &models.ValidationError{Field: "media_id", Message: "media_id is required"}
	}
	scope := localScopeFallback(mediaID)
	if s.scopes != nil {
		resolved, err := s.scopes.Resolve(ctx, mediaID)
		if err != nil {
			return "", err
		}
		scope = resolved
	}
	if !models.IsSharedGlossaryScope(scope) {
		return "", ErrGlossaryNotShareable
	}
	return scope, nil
}

func (s *GlossaryExchangeService) titleOf(ctx context.Context, mediaID string) string {
	if s.title == nil {
		return ""
	}
	return s.title(ctx, mediaID)
}

// Export builds the file for one title's glossary (default language only).
func (s *GlossaryExchangeService) Export(ctx context.Context, mediaID string) (*GlossaryExport, error) {
	scope, err := s.sharedScope(ctx, mediaID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListByScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	out := &GlossaryExport{
		Format: GlossaryExportFormat, Version: GlossaryExportVersion,
		ExportedAt: s.now().UTC(), Scope: scope, Title: s.titleOf(ctx, mediaID),
		Language: models.GlossaryDefaultLanguage, Terms: []GlossaryExportTerm{},
	}
	for _, r := range rows {
		lang := r.Language
		if lang == "" {
			lang = models.GlossaryDefaultLanguage
		}
		if lang != out.Language {
			continue
		}
		out.Terms = append(out.Terms, GlossaryExportTerm{TermSrc: r.TermSrc, TermZh: r.TermZh, Source: r.Source, Confirmed: r.Confirmed})
	}
	sort.SliceStable(out.Terms, func(i, j int) bool {
		return strings.ToLower(out.Terms[i].TermSrc) < strings.ToLower(out.Terms[j].TermSrc)
	})
	s.logger.Info("glossary exported", "scope", scope, "terms", len(out.Terms))
	return out, nil
}

// Import merges a file into the title's glossary. Insert-only: a term the
// title does not have yet is added as source=community, unconfirmed; a term
// it already has is NEVER overwritten — same rendering counts as skipped,
// a different one comes back as a conflict for the user to settle.
func (s *GlossaryExchangeService) Import(ctx context.Context, mediaID string, file *GlossaryExport) (*GlossaryImportResult, error) {
	if err := validateGlossaryFile(file); err != nil {
		return nil, err
	}
	scope, err := s.sharedScope(ctx, mediaID)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(file.Scope), scope) {
		return nil, fmt.Errorf("%w: file is %s, this title is %s", ErrGlossaryScopeMismatch, file.Scope, scope)
	}
	lang := file.Language
	if lang == "" {
		lang = models.GlossaryDefaultLanguage
	}
	rows, err := s.repo.ListByScope(ctx, scope)
	if err != nil {
		return nil, err
	}
	mine := make(map[string]models.GlossaryTerm, len(rows))
	for _, r := range rows {
		rl := r.Language
		if rl == "" {
			rl = models.GlossaryDefaultLanguage
		}
		if rl == lang {
			mine[strings.ToLower(strings.TrimSpace(r.TermSrc))] = r
		}
	}

	res := &GlossaryImportResult{Conflicts: []GlossaryImportConflict{}, Title: s.titleOf(ctx, mediaID)}
	seen := map[string]struct{}{}
	for _, t := range file.Terms {
		src, zh := strings.TrimSpace(t.TermSrc), strings.TrimSpace(t.TermZh)
		key := strings.ToLower(src)
		if src == "" || zh == "" {
			res.Skipped++
			continue
		}
		if _, dup := seen[key]; dup {
			res.Skipped++
			continue
		}
		seen[key] = struct{}{}
		if have, ok := mine[key]; ok {
			if strings.TrimSpace(have.TermZh) == zh {
				res.Skipped++
				continue
			}
			res.Conflicts = append(res.Conflicts, GlossaryImportConflict{
				ID: have.ID, TermSrc: have.TermSrc, Mine: have.TermZh, Theirs: zh,
				MineSource: have.Source, MineConfirmed: have.Confirmed,
			})
			continue
		}
		inserted, err := s.repo.InsertIfAbsent(ctx, &models.GlossaryTerm{
			MediaID: mediaID, Scope: scope, TermSrc: src, TermZh: zh, Language: lang,
			Source: models.GlossarySourceCommunity, Confirmed: false,
		})
		if err != nil {
			return nil, fmt.Errorf("import term %q: %w", src, err)
		}
		if inserted {
			res.Imported++
		} else {
			res.Skipped++ // raced with another writer; theirs did not land
		}
	}
	s.logger.Info("glossary imported", "scope", scope, "imported", res.Imported, "skipped", res.Skipped, "conflicts", len(res.Conflicts))
	return res, nil
}

func validateGlossaryFile(f *GlossaryExport) error {
	switch {
	case f == nil:
		return fmt.Errorf("%w: empty", ErrGlossaryImportInvalid)
	case f.Format != GlossaryExportFormat:
		return fmt.Errorf("%w: format %q", ErrGlossaryImportInvalid, f.Format)
	case f.Version != GlossaryExportVersion:
		return fmt.Errorf("%w: version %d is not supported (this Vido reads version %d)", ErrGlossaryImportInvalid, f.Version, GlossaryExportVersion)
	case strings.TrimSpace(f.Scope) == "":
		return fmt.Errorf("%w: no scope", ErrGlossaryImportInvalid)
	case len(f.Terms) > glossaryImportMaxTerms:
		return fmt.Errorf("%w: %d terms (limit %d)", ErrGlossaryImportInvalid, len(f.Terms), glossaryImportMaxTerms)
	}
	for _, t := range f.Terms {
		if utf8.RuneCountInString(t.TermSrc) > glossaryImportMaxRunes || utf8.RuneCountInString(t.TermZh) > glossaryImportMaxRunes {
			return fmt.Errorf("%w: a term is longer than %d characters", ErrGlossaryImportInvalid, glossaryImportMaxRunes)
		}
	}
	return nil
}
