package services

import "context"

// generationBatchIDKey carries the consent batch id down the item ctx
// (sub-7-6a). The pipeline reads it into subtitle_runs.batch_id so a batch
// receipt can be summed from the ledger after the fact — the same "ride the
// ctx, not the runner port" choice the shared Budget and the model id made.
type generationBatchIDKey struct{}

// WithGenerationBatchID stamps the batch id onto ctx; "" leaves ctx untouched.
func WithGenerationBatchID(ctx context.Context, batchID string) context.Context {
	if batchID == "" {
		return ctx
	}
	return context.WithValue(ctx, generationBatchIDKey{}, batchID)
}

// GenerationBatchIDFromContext returns the batch id, or "" outside a batch.
func GenerationBatchIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(generationBatchIDKey{}).(string); ok {
		return v
	}
	return ""
}
