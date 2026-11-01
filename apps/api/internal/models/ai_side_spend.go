package models

import "time"

// AISideSpendKind names which background AI call a side-spend row paid for.
// Closed set — mirrored by the CHECK in migration 047 (049 added burned_in_check).
type AISideSpendKind string

const (
	// AISideSpendKindParse is generic AI filename parsing.
	AISideSpendKindParse AISideSpendKind = "parse"
	// AISideSpendKindFansub is the fansub-specific filename prompt.
	AISideSpendKindFansub AISideSpendKind = "fansub"
	// AISideSpendKindKeywords is alternative-search-keyword generation.
	AISideSpendKindKeywords AISideSpendKind = "keywords"
	// AISideSpendKindBurnedInCheck is the burned-in subtitle check: one image
	// of an episode's frames read by Claude (disc-2026-11-burned-in-subs-b1,
	// migration 049).
	AISideSpendKindBurnedInCheck AISideSpendKind = "burned_in_check"
)

// AISideSpend is one paid AI call made OUTSIDE a subtitle run — filename
// parsing, fansub parsing, keyword generation (backlog-ai-parse-reads-env-keys-only-a).
// These calls are metered but never capped (Alexyu 2026-10-21): the row exists
// so the money shows up in「本月 AI 花費」, not to stop a scan.
//
// Kept out of subtitle_runs on purpose: that table's media_id / media_type are
// NOT NULL and media_type is CHECKed to movie/series/episode (migration 030) —
// a filename parse has no library row yet.
type AISideSpend struct {
	ID           int64
	Kind         AISideSpendKind
	Provider     string
	ModelID      string
	InputTokens  int64
	OutputTokens int64
	SpentUSD     float64
	CreatedAt    time.Time
}
