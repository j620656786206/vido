package models

import (
	"strings"
	"time"
)

// SubtitleRunStatus is the lifecycle state of one subtitle-pipeline run
// (migration 030 CHECK enum). Deliberately separate from SubtitleStatus: this
// tracks a RUN, that tracks the MEDIA ITEM, and the two vocabularies must not
// be merged — a run can fail while the item stays 'not_searched'.
type SubtitleRunStatus string

const (
	// SubtitleRunPending — recorded but not yet started.
	SubtitleRunPending SubtitleRunStatus = "pending"
	// SubtitleRunRunning — in flight.
	SubtitleRunRunning SubtitleRunStatus = "running"
	// SubtitleRunCompleted — produced a placed subtitle. The only status
	// FindCompletedRun matches (the resume predicate).
	SubtitleRunCompleted SubtitleRunStatus = "completed"
	// SubtitleRunFailed — ended in an error; error_message carries the reason.
	SubtitleRunFailed SubtitleRunStatus = "failed"
	// SubtitleRunSkipped — deliberately not run (routed out).
	SubtitleRunSkipped SubtitleRunStatus = "skipped"
)

// Media grains a run can attach to — the INTERNAL table vocabulary, matching
// migration 030's CHECK. Deliberately NOT requests.media_type's TMDB
// 'movie'|'tv' (migration 027): provenance is keyed to our own rows, and
// subtitle_status lives on all three tables (migrations 018 + 025).
const (
	SubtitleRunMediaMovie   = "movie"
	SubtitleRunMediaSeries  = "series"
	SubtitleRunMediaEpisode = "episode"
)

// AllSubtitleRunStatuses is the authoritative value set, mirroring migration
// 030's CHECK. Extend it here and nowhere else.
func AllSubtitleRunStatuses() []SubtitleRunStatus {
	return []SubtitleRunStatus{
		SubtitleRunPending,
		SubtitleRunRunning,
		SubtitleRunCompleted,
		SubtitleRunFailed,
		SubtitleRunSkipped,
	}
}

// IsValid reports whether s is a known run status.
func (s SubtitleRunStatus) IsValid() bool {
	for _, known := range AllSubtitleRunStatuses() {
		if s == known {
			return true
		}
	}
	return false
}

// RunVersion is the identity of "which inputs produced this translation".
//
// [@contract-v1] (story sub-1-2 AC #4) — consumed by sub-1-5b, which composes
// the cue-grain cache key as hash(cue) + RunVersion (D4), and by sub-1-6.
// Changing a field name or the tuple's membership is a Rule 20 bump plus a
// downstream stale-mark.
//
// Every field is mandatory even when empty. Omitting prompt or model would be a
// silent-failure trap: changing the prompt and re-running would return the
// cached prior translation, so two variants would look identical and the M1
// pilot's comparison data would be invalid with nothing surfacing an error.
type RunVersion struct {
	// MetadataHash is the snapshot hash of the TMDb metadata injected as
	// translation context (FR26). Computed by sub-1-5, not here.
	MetadataHash string
	// GlossaryVersion is the deterministic hash of the glossary pairs actually
	// fed into the translation prompt (sub-5-5; "" = no glossary — including
	// every pre-sub-5-5 run, which fed none). Computed by
	// subtitle.GlossaryVersionHash per D4's cross-component note.
	GlossaryVersion string
	// PromptVersion is the P11 constant's value at run time. Defined by
	// sub-1-5a, which owns the prompt text it lives beside.
	PromptVersion string
	// ModelID is the model that produced the translation, e.g. "claude-haiku-4-5".
	ModelID string
}

// Equal reports tuple equality — the resume predicate. Any single differing
// field makes a prior run non-matching, which is exactly what stops a re-run
// with a bumped prompt from being silently skipped.
func (v RunVersion) Equal(other RunVersion) bool {
	return v.MetadataHash == other.MetadataHash &&
		v.GlossaryVersion == other.GlossaryVersion &&
		v.PromptVersion == other.PromptVersion &&
		v.ModelID == other.ModelID
}

// SubtitleRun is one item-grain provenance record: which inputs produced one
// subtitle file (architecture D2, migration 030).
//
// Item grain lives HERE; cue grain lives in the tiered cache keyed by
// hash(source cue) + RunVersion (D4, sub-1-5). Building a per-cue table would
// be two storage mechanisms for one concept, which D2 explicitly rejects.
type SubtitleRun struct {
	ID              string            `db:"id" json:"id"`
	MediaID         string            `db:"media_id" json:"media_id"`
	MediaType       string            `db:"media_type" json:"media_type"`
	TMDbID          *int64            `db:"tmdb_id" json:"tmdb_id,omitempty"`
	MetadataHash    string            `db:"metadata_hash" json:"metadata_hash"`
	GlossaryVersion string            `db:"glossary_version" json:"glossary_version"`
	PromptVersion   string            `db:"prompt_version" json:"prompt_version"`
	ModelID         string            `db:"model_id" json:"model_id"`
	Status          SubtitleRunStatus `db:"status" json:"status"`
	SourceLanguage  string            `db:"source_language" json:"source_language,omitempty"`
	OutputPath      string            `db:"output_path" json:"output_path,omitempty"`
	CueCount        int               `db:"cue_count" json:"cue_count,omitempty"`
	CacheEnabled    bool              `db:"cache_enabled" json:"cache_enabled"`
	ErrorMessage    string            `db:"error_message" json:"error_message,omitempty"`
	StartedAt       time.Time         `db:"started_at" json:"started_at"`
	CompletedAt     *time.Time        `db:"completed_at" json:"completed_at,omitempty"`
	// SpentUSD/BudgetUSD are this run's OWN ai.Budget delta and ceiling,
	// stamped at every terminal transition (ux3-1-6, migration 032). Pointers
	// on purpose: NULL means "recorded before migration 032 / legacy path" —
	// absent is not $0, and home-summary skips NULL rows when resolving the
	// latest-spend readout.
	SpentUSD  *float64 `db:"spent_usd" json:"spent_usd,omitempty"`
	BudgetUSD *float64 `db:"budget_usd" json:"budget_usd,omitempty"`
	// StubbornCount is how many cues of a completed run shipped with their
	// ENGLISH original — quality-gate stubborn (FR16, ≤5%) plus, since
	// sub-6-2, the cues of a chunk whose request failed transiently through
	// every retry (≤20% together). TransientCount is that second population
	// on its own: the cues that are English because the provider was
	// unreachable, not because the model could not translate them. A run
	// with TransientCount > 0 is a PARTIAL delivery — the pre-flight lets the
	// item run again (its translated cues are cached; only the English ones
	// are re-sent) and the media row is left where the run found it rather
	// than marked `found`.
	//
	// Both are stamped at the completed transition of the extract→translate
	// routes only. NULL = not counted: a run recorded before migration 034,
	// or the ASR route, whose keep-English tolerance lives inside the
	// transcription service and is not measured here. Absent is not 0.
	StubbornCount  *int `db:"stubborn_count" json:"stubborn_count,omitempty"`
	TransientCount *int `db:"transient_count" json:"transient_count,omitempty"`

	// Ledger columns (sub-7-6a, migration 041). Route is the lane the item
	// took (SubtitleRunRoute*); "" = recorded before the column existed or
	// failed before routing. CacheHitCues is how many cues the segment cache
	// served instead of the model — measured on the translate lanes only,
	// NULL elsewhere (absent is not 0). BatchID ties the run to the consent
	// batch it ran in; "" for solo / pool runs.
	Route        string `db:"route" json:"route,omitempty"`
	CacheHitCues *int   `db:"cache_hit_cues" json:"cache_hit_cues,omitempty"`
	BatchID      string `db:"batch_id" json:"batch_id,omitempty"`

	// TriggeredBy (infra-optin-usage-report-a1, migration 043) is who started
	// the run: SubtitleRunTriggeredAuto (AutoGenerator after a scan, or the
	// request-completion trigger after a download) or SubtitleRunTriggeredManual
	// (a user action). "" = recorded before the column existed — not guessed.
	TriggeredBy string `db:"triggered_by" json:"triggered_by,omitempty"`
}

// Who started a run (SubtitleRun.TriggeredBy). The opt-in usage report counts
// only "auto" — subtitles Vido produced without anyone asking.
const (
	SubtitleRunTriggeredAuto   = "auto"
	SubtitleRunTriggeredManual = "manual"
)

// The route vocabulary the ledger stores. The first five mirror
// subtitle.RouteKind byte for byte (the pipeline writes string(decision.Kind));
// "asr" is the lane the pipeline's no-text-source fallback and the Route C
// transcription engine share; "online" is the online-provider engine. Kept as strings, not a typed enum, so the two
// packages that write them (subtitle, services) need no import of each other.
const (
	SubtitleRunRouteDeliverDirect      = "deliver_direct"
	SubtitleRunRouteConvertThenDeliver = "convert_then_deliver"
	SubtitleRunRouteTranslate          = "translate"
	SubtitleRunRouteSkip               = "skip"
	SubtitleRunRouteNoTextSource       = "no_text_source"
	SubtitleRunRouteASR                = "asr"
	// SubtitleRunRouteOnline is the online-provider lane (Assrt / Zimuku /
	// OpenSubtitles via subtitle.Engine): an existing Traditional Chinese
	// subtitle was found, downloaded and placed. It does not mirror a
	// RouteKind — the engine is not the pipeline (infra-optin-usage-report-a1).
	SubtitleRunRouteOnline = "online"
)

// ReceiptPayload is the `subtitle_run_receipt` SSE body (sub-7-6a
// [@contract-v1]): everything a "本次 $0.53 · claude-sonnet-5 · 844 句 ·
// cache 命中 12%" line needs, emitted once per run at its terminal write by
// BOTH legs (the pipeline and the Route C transcription engine) so the
// frontend has one listener. Optional facts are omitted rather than zeroed:
// spent_usd/budget_usd are absent when no Budget was on the ctx, cache_hit_cues
// when the lane does not measure it, batch_id / route when unknown.
func (r *SubtitleRun) ReceiptPayload() map[string]interface{} {
	data := map[string]interface{}{
		"run_id":     r.ID,
		"media_id":   r.MediaID,
		"media_type": r.MediaType,
		"status":     string(r.Status),
		"model_id":   r.ModelID,
		"cue_count":  r.CueCount,
	}
	if r.Route != "" {
		data["route"] = r.Route
	}
	if r.BatchID != "" {
		data["batch_id"] = r.BatchID
	}
	if r.CacheHitCues != nil {
		data["cache_hit_cues"] = *r.CacheHitCues
	}
	if r.SpentUSD != nil {
		data["spent_usd"] = *r.SpentUSD
	}
	if r.BudgetUSD != nil {
		data["budget_usd"] = *r.BudgetUSD
	}
	if r.CompletedAt != nil {
		data["completed_at"] = r.CompletedAt.UTC()
	}
	return data
}

// Validate checks the caller-supplied fields of a run before it is persisted.
// The DB CHECKs are the backstop; this returns a typed ValidationError so the
// caller sees which field is wrong rather than a driver constraint message.
func (r *SubtitleRun) Validate() error {
	if strings.TrimSpace(r.MediaID) == "" {
		return &ValidationError{Field: "media_id", Message: "media_id is required"}
	}
	switch r.MediaType {
	case SubtitleRunMediaMovie, SubtitleRunMediaSeries, SubtitleRunMediaEpisode:
	default:
		return &ValidationError{Field: "media_type", Message: "media_type must be 'movie', 'series', or 'episode'"}
	}
	if r.Status != "" && !r.Status.IsValid() {
		return &ValidationError{Field: "status", Message: "status must be 'pending', 'running', 'completed', 'failed', or 'skipped'"}
	}
	return nil
}

// Version returns the run's version tuple — the value sub-1-5b hashes into the
// cue-grain cache key and the repository matches on when resuming.
func (r *SubtitleRun) Version() RunVersion {
	return RunVersion{
		MetadataHash:    r.MetadataHash,
		GlossaryVersion: r.GlossaryVersion,
		PromptVersion:   r.PromptVersion,
		ModelID:         r.ModelID,
	}
}
