package main

import (
	"context"
	"sync"
	"time"

	"github.com/vido/api/internal/ai"
)

// pacedCompleter spaces calls to the model under test at least `gap` apart.
//
// Free tiers are rate-limited per minute (Gemini's free tier allows 5 requests
// a minute on gemini-3.8-flash, 2026-10-14) and the translation stack's own
// retry waits only 1–2 s, so an unpaced run spends most of its requests on
// 429s and keeps English for 150 of 200 cues — a measurement of the quota,
// not of the model. The judge (Claude) is not paced.
type pacedCompleter struct {
	inner ai.TextCompleter
	gap   time.Duration
	sleep func(context.Context, time.Duration) error

	mu   sync.Mutex
	last time.Time
}

func newPacedCompleter(inner ai.TextCompleter, gap time.Duration) ai.TextCompleter {
	if gap <= 0 {
		return inner
	}
	return &pacedCompleter{inner: inner, gap: gap, sleep: sleepCtx}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (p *pacedCompleter) CompleteText(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	p.mu.Lock()
	wait := time.Until(p.last.Add(p.gap))
	p.last = time.Now()
	if wait > 0 {
		p.last = p.last.Add(wait)
	}
	p.mu.Unlock()
	if wait > 0 {
		if err := p.sleep(ctx, wait); err != nil {
			return "", err
		}
	}
	return p.inner.CompleteText(ctx, systemPrompt, userPrompt, maxTokens)
}
