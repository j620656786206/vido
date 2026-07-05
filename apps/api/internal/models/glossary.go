package models

import (
	"strings"
	"time"
)

// Glossary term sources (migration 028 CHECK enum). Records where a term↔zh
// mapping came from so the F6 review UI can show provenance.
const (
	// GlossarySourceSubtitle — mined from a subtitle during generation.
	GlossarySourceSubtitle = "subtitle"
	// GlossarySourceMetadata — mined from show metadata (cast/character table).
	GlossarySourceMetadata = "metadata"
	// GlossarySourceManual — entered/edited by the user.
	GlossarySourceManual = "manual"
)

// GlossaryDefaultLanguage is the target rendering language for a glossary term.
const GlossaryDefaultLanguage = "zh-Hant"

// GlossaryTerm is one per-show proper-noun mapping (Story 9R-6). The glossary
// is the Route C keystone: it fixes proper-noun drift across generation runs
// and is shared by subtitle translation (9R-7) and .nfo localization (9R-13).
type GlossaryTerm struct {
	ID        string    `db:"id" json:"id"`
	MediaID   string    `db:"media_id" json:"media_id"`
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
	if strings.TrimSpace(g.TermSrc) == "" {
		return &ValidationError{Field: "term_src", Message: "term_src is required"}
	}
	if strings.TrimSpace(g.TermZh) == "" {
		return &ValidationError{Field: "term_zh", Message: "term_zh is required"}
	}
	if s := g.Source; s != "" && s != GlossarySourceSubtitle && s != GlossarySourceMetadata && s != GlossarySourceManual {
		return &ValidationError{Field: "source", Message: "source must be 'subtitle', 'metadata', or 'manual'"}
	}
	return nil
}
