package ai

import (
	"context"
	"errors"
)

// disc-2026-10-translation-batch-truncation: Sonnet 5 thinks by default
// (adaptive thinking), and the thinking tokens count against max_tokens. On
// See S01E04 one 50-cue translation batch thought past the 4096 cap and came
// back with no text at all — the whole episode failed. Measured on that batch:
// default 1,003–3,597 output tokens (the translation itself is ~1,000),
// thinking disabled 1,000–1,033.

// ErrAIResponseTruncated marks a reply that hit max_tokens before it carried
// any usable text. It is always returned wrapped together with
// ErrAIInvalidResponse, so existing errors.Is(err, ErrAIInvalidResponse)
// checks keep matching; callers that can retry with a smaller request (the
// subtitle translator splits the batch) test for this one.
var ErrAIResponseTruncated = errors.New("AI_RESPONSE_TRUNCATED: reply hit the max_tokens cap before any text")

type thinkingOffKey struct{}

// WithThinkingOff asks the provider not to spend reasoning tokens on requests
// made under ctx. Claude sends thinking {type: disabled}; providers without a
// thinking switch ignore it.
func WithThinkingOff(ctx context.Context) context.Context {
	return context.WithValue(ctx, thinkingOffKey{}, true)
}

// ThinkingOffFromContext reports whether WithThinkingOff is set on ctx.
func ThinkingOffFromContext(ctx context.Context) bool {
	off, _ := ctx.Value(thinkingOffKey{}).(bool)
	return off
}
