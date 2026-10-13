package services

// disc-2026-10-batch-stop-on-credit-exhausted-a — an AI account with no money
// left stops the whole batch; the rest waits, paused, for a top-up.

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

func TestGenerationBatch_EmptyAccountStopsTheBatch(t *testing.T) {
	credit := &ai.CreditExhaustedError{Provider: "openai", Code: "insufficient_quota", BillingURL: ai.OpenAIBillingURL}
	runner := &fakeGenerationRunner{
		available: true,
		// Wrapped the way the pipeline wraps it on the way up.
		errs: map[string]error{uuidB: fmt.Errorf("transcribe: %w", fmt.Errorf("%w: %w", ai.ErrWhisperAPIError, credit))},
	}
	p, client := newTestGenerationProcessor(t, runner, threeMovies(), 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)

	assert.Equal(t, []string{uuidA, uuidB}, runner.callIDs(), "C would fail the same way — never started")

	events := eventsUntilTerminal(t, client)
	require.NotEmpty(t, events)
	last := events[len(events)-1]
	assert.Equal(t, GenerationBatchStatusCreditExhausted, last["status"])
	assert.Equal(t, 1, last["success_count"])
	assert.Equal(t, 0, last["fail_count"], "no credit is not the file's fault")
	assert.Equal(t, 2, last["paused_count"], "the item that hit it + the rest of the queue")
	assert.Equal(t, "openai", last["credit_provider"])
	assert.Equal(t, ai.OpenAIBillingURL, last["billing_url"])

	_, kept := p.Snapshot()
	require.NotNil(t, kept)
	assert.Equal(t, GenerationBatchStatusCreditExhausted, kept.Status)
	assert.Equal(t, "openai", kept.CreditProvider)
	assert.Equal(t, ai.OpenAIBillingURL, kept.BillingURL)
	statuses := map[string]GenerationBatchItemStatus{}
	for _, it := range kept.Items {
		statuses[it.MediaID] = it.Status
	}
	assert.Equal(t, map[string]GenerationBatchItemStatus{
		uuidA: GenerationBatchItemDone, uuidB: GenerationBatchItemPaused, uuidC: GenerationBatchItemPaused,
	}, statuses)
}

// A bare quota error may be a rate limit (Gemini, Claude 429): that item
// fails and the batch goes on, exactly as before.
func TestGenerationBatch_BareQuotaErrorStillOnlyFailsThatItem(t *testing.T) {
	runner := &fakeGenerationRunner{
		available: true,
		errs:      map[string]error{uuidB: fmt.Errorf("translate: %w", ai.ErrAIQuotaExceeded)},
	}
	p, client := newTestGenerationProcessor(t, runner, threeMovies(), 5)

	_, _, err := p.Start(context.Background(), "missing", nil, 0, "")
	require.NoError(t, err)
	waitUntilIdle(t, p)

	assert.Equal(t, []string{uuidA, uuidB, uuidC}, runner.callIDs())
	events := eventsUntilTerminal(t, client)
	last := events[len(events)-1]
	assert.Equal(t, GenerationBatchStatusComplete, last["status"])
	assert.Equal(t, 1, last["fail_count"])
	assert.Equal(t, "", last["credit_provider"])
	assert.Equal(t, "", last["billing_url"])
}
