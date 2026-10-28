package services

import (
	"sync"

	"github.com/vido/api/internal/ai"
)

// SoloSpendBoard is the ai.Budget of every running single-title generation
// job, by media id (disc-2026-10-solo-live-spend). A pipeline-mode click
// (subtitle.SoloRunner) owns its job's Budget, but most of the job's progress
// events are sent by code that does not own it: the pipeline's
// subtitle_progress stages and the ASR leg's transcription_* events. Both look
// the job up here, so every one of those events can carry what the job has
// spent so far.
//
// Only solo jobs are tracked. A batch item spends from the batch's shared
// envelope and a pool item from its own per-item one; neither is on the
// board, so their events keep carrying no cost keys (absent ≠ 0).
//
// Media ids are UUIDs (movie or episode), and SoloGuard keeps one generation
// per item at a time, so one id never names two running jobs.
//
// A nil *SoloSpendBoard is valid: Track/Untrack do nothing and Lookup finds
// nothing.
type SoloSpendBoard struct {
	mu      sync.Mutex
	byMedia map[string]*ai.Budget
}

// NewSoloSpendBoard returns an empty board.
func NewSoloSpendBoard() *SoloSpendBoard {
	return &SoloSpendBoard{byMedia: make(map[string]*ai.Budget)}
}

// Track records budget as the running solo job of mediaID.
func (b *SoloSpendBoard) Track(mediaID string, budget *ai.Budget) {
	if b == nil || mediaID == "" || budget == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.byMedia[mediaID] = budget
}

// Untrack forgets mediaID — only while it still names budget, so a job that
// ends late can never erase the next job's entry.
func (b *SoloSpendBoard) Untrack(mediaID string, budget *ai.Budget) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if cur, ok := b.byMedia[mediaID]; ok && cur == budget {
		delete(b.byMedia, mediaID)
	}
}

// Lookup returns the running solo job's Budget for mediaID, or nil.
func (b *SoloSpendBoard) Lookup(mediaID string) *ai.Budget {
	if b == nil || mediaID == "" {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.byMedia[mediaID]
}

// AddRunCost writes the 9R-17 cost keys of one job's Budget into an event
// payload: spent_usd always (the raw recorded figure — the client rounds),
// budget_usd only when a ceiling is set (absent = no ceiling, never 0). A nil
// budget writes nothing. The one writer of these keys for single-title runs,
// so the solo runner, the ASR leg and the pipeline stages cannot drift.
func AddRunCost(data map[string]interface{}, budget *ai.Budget) {
	if data == nil || budget == nil {
		return
	}
	snap := budget.Snapshot()
	data["spent_usd"] = snap.SpentUSD
	if snap.BudgetUSD > 0 {
		data["budget_usd"] = snap.BudgetUSD
	}
}
