package models

import (
	"strconv"
	"strings"
	"time"
)

// Glossary term sources. Records where a term↔zh mapping came from so the F6
// review UI can show provenance. Since migration 036 the set is enforced HERE
// (GlossaryTerm.Validate), not by a SQLite CHECK — SQLite cannot ALTER a CHECK,
// and every new provenance (sub-7-5 official subtitles, sub-8-1 community
// imports) would otherwise cost a table rebuild.
const (
	// GlossarySourceSubtitle — mined from a subtitle during generation.
	GlossarySourceSubtitle = "subtitle"
	// GlossarySourceMetadata — mined from show metadata (cast/character table).
	GlossarySourceMetadata = "metadata"
	// GlossarySourceManual — entered/edited by the user.
	GlossarySourceManual = "manual"
	// GlossarySourceOfficialSubtitle — aligned out of an official zh-Hant
	// subtitle of the same show (sub-7-5); the most trusted machine source.
	GlossarySourceOfficialSubtitle = "official_subtitle"
	// GlossarySourceCommunity — imported from another install (sub-8-1).
	GlossarySourceCommunity = "community"
)

// GlossarySources is the closed set Validate accepts, in display order.
var GlossarySources = []string{
	GlossarySourceSubtitle,
	GlossarySourceMetadata,
	GlossarySourceManual,
	GlossarySourceOfficialSubtitle,
	GlossarySourceCommunity,
}

// IsGlossarySource reports whether s is one of GlossarySources.
func IsGlossarySource(s string) bool {
	for _, known := range GlossarySources {
		if s == known {
			return true
		}
	}
	return false
}

// GlossaryDefaultLanguage is the target rendering language for a glossary term.
const GlossaryDefaultLanguage = "zh-Hant"

// Glossary scope prefixes (sub-7-1). A scope names the DRAWER a term lives in.
// It is a namespaced string on purpose — not a `tmdb_id INTEGER` column — so
// another id source (IMDb, TVDB, a collection, a global lexicon) is a new
// prefix and a resolver change, never a table rebuild (eval-1 彈性檢查).
//
//	tmdb:tv:<id>      a series matched to TMDb — shared by every copy of the show
//	tmdb:movie:<id>   a movie matched to TMDb
//	local:<media_id>  not matched (yet): keyed by this machine's row id, the
//	                  pre-sub-7-1 behaviour; upgraded in place once a match lands
const (
	GlossaryScopePrefixTMDbTV    = "tmdb:tv:"
	GlossaryScopePrefixTMDbMovie = "tmdb:movie:"
	GlossaryScopePrefixLocal     = "local:"
)

// GlossaryScopeTV builds the shared scope for a TMDb series id.
func GlossaryScopeTV(tmdbID int64) string {
	return GlossaryScopePrefixTMDbTV + strconv.FormatInt(tmdbID, 10)
}

// GlossaryScopeMovie builds the shared scope for a TMDb movie id.
func GlossaryScopeMovie(tmdbID int64) string {
	return GlossaryScopePrefixTMDbMovie + strconv.FormatInt(tmdbID, 10)
}

// GlossaryScopeLocal builds the fallback scope for an unmatched local media id.
func GlossaryScopeLocal(mediaID string) string {
	return GlossaryScopePrefixLocal + strings.TrimSpace(mediaID)
}

// ParseSharedGlossaryScope splits a shared scope back into the media kind
// ("tv" / "movie") and TMDb id it was built from. ok=false for `local:` and
// anything malformed.
func ParseSharedGlossaryScope(scope string) (kind string, tmdbID int64, ok bool) {
	var rest string
	switch {
	case strings.HasPrefix(scope, GlossaryScopePrefixTMDbTV):
		kind, rest = "tv", strings.TrimPrefix(scope, GlossaryScopePrefixTMDbTV)
	case strings.HasPrefix(scope, GlossaryScopePrefixTMDbMovie):
		kind, rest = "movie", strings.TrimPrefix(scope, GlossaryScopePrefixTMDbMovie)
	default:
		return "", 0, false
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || id <= 0 {
		return "", 0, false
	}
	return kind, id, true
}

// IsLocalGlossaryScope reports whether scope is a `local:` fallback — terms that
// can never be shared because nothing outside this machine knows the key.
func IsLocalGlossaryScope(scope string) bool {
	return strings.HasPrefix(scope, GlossaryScopePrefixLocal)
}

// IsSharedGlossaryScope reports whether scope is keyed by a world-wide id.
func IsSharedGlossaryScope(scope string) bool {
	return strings.HasPrefix(scope, GlossaryScopePrefixTMDbTV) ||
		strings.HasPrefix(scope, GlossaryScopePrefixTMDbMovie)
}

// GlossaryTerm is one per-show proper-noun mapping (Story 9R-6). The glossary
// is the Route C keystone: it fixes proper-noun drift across generation runs
// and is shared by subtitle translation (9R-7) and .nfo localization (9R-13).
//
// Scope (sub-7-1) is the key the term is READ by; MediaID is the local row
// that WROTE it, kept for audit until the next migration drops it (Rule 24
// superseded-mechanism corollary).
type GlossaryTerm struct {
	ID        string    `db:"id" json:"id"`
	MediaID   string    `db:"media_id" json:"media_id"`
	Scope     string    `db:"scope" json:"scope"`
	TermSrc   string    `db:"term_src" json:"term_src"`
	TermZh    string    `db:"term_zh" json:"term_zh"`
	Language  string    `db:"language" json:"language"`
	Source    string    `db:"source" json:"source"`
	Confirmed bool      `db:"confirmed" json:"confirmed"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Validate checks the client/caller-supplied fields of a glossary term.
func (g *GlossaryTerm) Validate() error {
	if strings.TrimSpace(g.MediaID) == "" {
		return &ValidationError{Field: "media_id", Message: "media_id is required"}
	}
	if strings.TrimSpace(g.Scope) == "" {
		return &ValidationError{Field: "scope", Message: "scope is required — resolve the media id first"}
	}
	if strings.TrimSpace(g.TermSrc) == "" {
		return &ValidationError{Field: "term_src", Message: "term_src is required"}
	}
	if strings.TrimSpace(g.TermZh) == "" {
		return &ValidationError{Field: "term_zh", Message: "term_zh is required"}
	}
	if s := g.Source; s != "" && !IsGlossarySource(s) {
		return &ValidationError{Field: "source", Message: "source must be one of: " + strings.Join(GlossarySources, ", ")}
	}
	return nil
}
