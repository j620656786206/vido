package ai

import "context"

// The per-run model choice travels on the ctx (sub-6-8a AC #5).
//
// It is a ctx value for the same reason the per-run Budget is one: every layer
// between the API boundary and the provider — the batch processor, the worker
// pool, the pipeline, the translation service — would otherwise have to grow a
// model parameter it does nothing with, and two of those seams are stamped
// contracts (`ChunkTranslator`, `TranslateContext`) that a widening would bump
// for no behavioural gain. The provider holder is the only reader; everything
// in between just carries the ctx it already carries.
//
// Absent (or empty) means "the deployment's effective default" — never a
// silent fallback to some other model.
type modelIDKey struct{}

// WithModelID returns a ctx that pins one model for every AI call made under
// it. An empty id leaves the ctx unchanged, so a caller can pass through an
// unset choice without special-casing it.
func WithModelID(ctx context.Context, model string) context.Context {
	if model == "" {
		return ctx
	}
	return context.WithValue(ctx, modelIDKey{}, model)
}

// ModelIDFromContext returns the pinned model, or "" when the caller did not
// choose one.
func ModelIDFromContext(ctx context.Context) string {
	model, _ := ctx.Value(modelIDKey{}).(string)
	return model
}
