package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/vido/api/internal/ai"
)

// BatchUsageReader is the one GenerationBatchProcessor method GET /ai/usage
// needs (Rule 11).
type BatchUsageReader interface {
	ActiveSnapshot() (ai.BudgetSnapshot, bool)
}

// ManualUsageReader is the one TranscriptionService method GET /ai/usage
// needs (Rule 11).
type ManualUsageReader interface {
	ActiveManualUsage() (spentUSD, budgetUSD float64, ok bool)
}

// AIUsageHandler serves GET /api/v1/ai/usage (Story 9R-17 AC #3): ONE place
// that answers "what is AI spending right now", whether the paid run is a
// generation batch or a single solo transcription. The batch status route
// only knows batches.
type AIUsageHandler struct {
	batch  BatchUsageReader
	manual ManualUsageReader
}

// NewAIUsageHandler builds the handler; either reader may be nil (unwired).
func NewAIUsageHandler(batch BatchUsageReader, manual ManualUsageReader) *AIUsageHandler {
	return &AIUsageHandler{batch: batch, manual: manual}
}

// RegisterRoutes mounts GET /ai/usage on the API group.
func (h *AIUsageHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/ai/usage", h.GetUsage)
}

// AIActiveRun is the live spend of the paid run in progress.
type AIActiveRun struct {
	// Kind is "batch" or "manual".
	Kind      string  `json:"kind"`
	SpentUSD  float64 `json:"spent_usd"`
	BudgetUSD float64 `json:"budget_usd"`
	// RemainingUSD is budget − spent, floored at 0; 0 when no ceiling is set
	// (BudgetUSD 0 means unlimited).
	RemainingUSD float64 `json:"remaining_usd"`
}

// AIUsageResponse is the GET /ai/usage data. ActiveRun is null when nothing
// paid is running — idle is a normal state, not a 404. No started_at: neither
// side records one, and this endpoint does not invent it.
type AIUsageResponse struct {
	ActiveRun *AIActiveRun `json:"active_run"`
}

// GetUsage handles GET /api/v1/ai/usage. A running batch is reported in
// preference to solo runs (one active_run; a finer split is future scope).
// @Summary Live AI spend of the paid run in progress (batch or solo)
// @Tags ai
// @Produce json
// @Success 200 {object} APIResponse{data=AIUsageResponse}
// @Router /api/v1/ai/usage [get]
func (h *AIUsageHandler) GetUsage(c *gin.Context) {
	if h.batch != nil {
		if snap, ok := h.batch.ActiveSnapshot(); ok {
			SuccessResponse(c, AIUsageResponse{ActiveRun: activeRun("batch", snap.SpentUSD, snap.BudgetUSD)})
			return
		}
	}
	if h.manual != nil {
		if spent, budget, ok := h.manual.ActiveManualUsage(); ok {
			SuccessResponse(c, AIUsageResponse{ActiveRun: activeRun("manual", spent, budget)})
			return
		}
	}
	SuccessResponse(c, AIUsageResponse{ActiveRun: nil})
}

func activeRun(kind string, spent, budget float64) *AIActiveRun {
	remaining := 0.0
	if budget > 0 && budget > spent {
		remaining = budget - spent
	}
	return &AIActiveRun{Kind: kind, SpentUSD: spent, BudgetUSD: budget, RemainingUSD: remaining}
}
