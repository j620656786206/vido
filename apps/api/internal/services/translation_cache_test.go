package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/segkey"
)

// ─── disc-2026-09-generation-resume-a-translation-cache AC #3/#4/#5 ────────
//
// A 1,910-cue film costs ~$2.71 and 191 batches to translate. Until this story,
// a batch that hit the money ceiling at cue 1,200 threw away all 1,200 paid-for
// translations: the next run started at cue 1 and hit the same ceiling in the
// same place, forever. Each batch is now written to the shared segment cache as
// soon as it comes back, and the next run translates only what is missing.

// countingCompleter answers each batch by echoing back the indices it was
// given, and can fail at a chosen call — the budget ceiling's shape.
type countingCompleter struct {
	mu       sync.Mutex
	calls    int
	prompts  []string
	failAt   int   // 1-based call that returns ErrBudgetExceeded; 0 = never
	failWith error // overrides the budget sentinel when set
	prefix   string
}

func (c *countingCompleter) CompleteText(_ context.Context, _, userPrompt string, _ int) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	c.prompts = append(c.prompts, userPrompt)
	if c.failAt > 0 && c.calls == c.failAt {
		if c.failWith != nil {
			return "", c.failWith
		}
		return "", ai.ErrBudgetExceeded
	}
	var out strings.Builder
	for _, idx := range promptIndices(userPrompt) {
		fmt.Fprintf(&out, "[%d] %s譯文%d\n", idx, c.prefix, idx)
	}
	return out.String(), nil
}

func (c *countingCompleter) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

// promptIndices pulls the block indices out of the rendered user prompt — the
// same `[N] text` shape the parser reads back, so the fake answers exactly the
// batch it was asked about (and never the context lines, which the real model
// is told not to translate).
func promptIndices(userPrompt string) []int {
	var out []int
	body := userPrompt
	if i := strings.LastIndex(body, "Previous context"); i >= 0 {
		// Context section comes first; translate only what follows it.
		if j := strings.Index(body[i:], "\n\n"); j >= 0 {
			body = body[i+j:]
		}
	}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[") {
			continue
		}
		end := strings.Index(line, "]")
		if end < 0 {
			continue
		}
		var idx int
		if _, err := fmt.Sscanf(line[1:end], "%d", &idx); err == nil {
			out = append(out, idx)
		}
	}
	return out
}

// memoryStore is the in-package SegmentStore fake: two methods, failure
// injection on each, and a ledger of what was written.
type memoryStore struct {
	mu      sync.Mutex
	entries map[string]string
	writes  []string
	ttls    []time.Duration
	getErr  error
	setErr  error
	getKeys int
	getMany int
}

func newMemoryStore() *memoryStore { return &memoryStore{entries: map[string]string{}} }

func (m *memoryStore) GetMany(_ context.Context, keys []string) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getMany++
	m.getKeys += len(keys)
	if m.getErr != nil {
		return nil, m.getErr
	}
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := m.entries[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

func (m *memoryStore) Set(_ context.Context, key, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.setErr != nil {
		return m.setErr
	}
	m.entries[key] = value
	m.writes = append(m.writes, key)
	m.ttls = append(m.ttls, ttl)
	return nil
}

func (m *memoryStore) size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.entries)
}

func cacheTestBlocks(n int) []TranslationBlock {
	blocks := make([]TranslationBlock, n)
	for i := range blocks {
		blocks[i] = TranslationBlock{
			Index: i + 1,
			Start: fmt.Sprintf("00:%02d:%02d,000", i/60, i%60),
			End:   fmt.Sprintf("00:%02d:%02d,000", (i+1)/60, (i+1)%60),
			Text:  fmt.Sprintf("Line %d", i+1),
		}
	}
	return blocks
}

func testRunVersion() models.RunVersion {
	return models.RunVersion{MetadataHash: "m", GlossaryVersion: "g", PromptVersion: "p", ModelID: "claude-sonnet-5"}
}

// ─── AC #3: ModelID is never empty ─────────────────────────────────────────

type modelNamingCompleter struct {
	countingCompleter
	model string
}

func (m *modelNamingCompleter) EffectiveModel() string { return m.model }

func TestEffectiveModelID_ThreeFallbacks(t *testing.T) {
	t.Run("the run's explicit model wins", func(t *testing.T) {
		svc := NewTranslationService(&modelNamingCompleter{model: "claude-haiku-4-5"}, nil)
		ctx := ai.WithModelID(context.Background(), "claude-opus-5")
		assert.Equal(t, "claude-opus-5", svc.EffectiveModelID(ctx))
	})

	t.Run("otherwise the provider names itself", func(t *testing.T) {
		svc := NewTranslationService(&modelNamingCompleter{model: "claude-haiku-4-5"}, nil)
		assert.Equal(t, "claude-haiku-4-5", svc.EffectiveModelID(context.Background()))
	})

	t.Run("a provider that cannot name itself falls back to the deployment default", func(t *testing.T) {
		svc := NewTranslationService(&countingCompleter{}, nil)
		assert.Equal(t, ai.DefaultClaudeModel, svc.EffectiveModelID(context.Background()))
		assert.NotEmpty(t, ai.DefaultClaudeModel,
			"an empty ModelID would key the default-model runs differently from the same model picked explicitly, halving every hit rate")
	})

	t.Run("an empty provider answer still falls back", func(t *testing.T) {
		svc := NewTranslationService(&modelNamingCompleter{model: ""}, nil)
		assert.Equal(t, ai.DefaultClaudeModel, svc.EffectiveModelID(context.Background()))
	})
}

// ─── AC #4/#5: resume ──────────────────────────────────────────────────────

func TestTranslate_ResumesFromTheCachedBatches(t *testing.T) {
	blocks := cacheTestBlocks(1910)
	store := newMemoryStore()
	version := testRunVersion()

	// First run: the ceiling trips on batch 121 (1,200 cues already paid for).
	first := &countingCompleter{failAt: 121}
	svc := NewTranslationService(first, nil)
	_, _, _, err := svc.TranslateWithGlossaryHarvest(context.Background(), blocks, nil, nil,
		WithSegmentCache(store, version))

	require.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrBudgetExceeded)
	assert.Equal(t, 1200, store.size(), "every cue the run paid for is remembered")
	assert.Equal(t, segkey.SegmentKey(blocks[0].Text, version), store.writes[0],
		"stored under the key the extract leg would use for the same cue and version")
	assert.Equal(t, segkey.TTL, store.ttls[0])

	// Second run: only the 710 unfinished cues reach the model.
	readsBefore := store.getMany
	second := &countingCompleter{}
	svc2 := NewTranslationService(second, nil)
	var progress []float64
	result, _, outcome, err := svc2.TranslateWithGlossaryHarvest(context.Background(), blocks, nil,
		func(pct float64) { progress = append(progress, pct) }, WithSegmentCache(store, version))

	require.NoError(t, err)
	assert.Equal(t, 71, second.count(), "1,910 − 1,200 = 710 cues = 71 batches")
	assert.Equal(t, 1, store.getMany-readsBefore, "one batched read for the whole film, never one per cue")
	assert.Equal(t, 1910, store.getKeys/store.getMany, "…covering every cue in one call")
	require.Len(t, result, 1910)
	assert.Equal(t, 0, outcome.EnglishKeptBlocks)
	for i, b := range result {
		assert.Equalf(t, fmt.Sprintf("譯文%d", i+1), b.Text, "cue %d", i+1)
		assert.Equal(t, blocks[i].Index, b.Index)
		assert.Equal(t, blocks[i].Start, b.Start)
		assert.Equal(t, blocks[i].End, b.End)
	}
	require.NotEmpty(t, progress)
	assert.GreaterOrEqual(t, progress[0], 62.0,
		"the first progress frame already reflects what an earlier run paid for")
	assert.Equal(t, 100.0, progress[len(progress)-1])

	// A third run pays nothing at all.
	third := &countingCompleter{}
	svc3 := NewTranslationService(third, nil)
	_, _, _, err = svc3.TranslateWithGlossaryHarvest(context.Background(), blocks, nil, nil,
		WithSegmentCache(store, version))
	require.NoError(t, err)
	assert.Equal(t, 0, third.count())
}

// The context window is what keeps a resumed translation reading like one
// piece: the first fresh batch must see the CACHED translations before it, not
// English and not the previous fresh batch.
func TestTranslate_ResumedBatchSeesCachedContext(t *testing.T) {
	blocks := cacheTestBlocks(20)
	store := newMemoryStore()
	version := testRunVersion()
	for i := 0; i < 10; i++ {
		require.NoError(t, store.Set(context.Background(),
			segkey.SegmentKey(blocks[i].Text, version), fmt.Sprintf("舊譯文%d", i+1), segkey.TTL))
	}

	completer := &countingCompleter{}
	svc := NewTranslationService(completer, nil)
	_, _, _, err := svc.TranslateWithGlossaryHarvest(context.Background(), blocks, nil, nil,
		WithSegmentCache(store, version))

	require.NoError(t, err)
	require.Equal(t, 1, completer.count())
	prompt := completer.prompts[0]
	assert.Contains(t, prompt, "Previous context")
	assert.Contains(t, prompt, "舊譯文10", "the cue right before the resume point is the cached TRANSLATION")
	assert.NotContains(t, prompt, "Line 10\n", "…not the English it replaced")
}

func TestTranslate_CacheFailuresNeverFailThePaidRun(t *testing.T) {
	blocks := cacheTestBlocks(30)
	version := testRunVersion()

	t.Run("read failure → everything is translated, run succeeds", func(t *testing.T) {
		store := newMemoryStore()
		store.getErr = errors.New("db locked")
		completer := &countingCompleter{}
		svc := NewTranslationService(completer, nil)
		result, _, _, err := svc.TranslateWithGlossaryHarvest(context.Background(), blocks, nil, nil,
			WithSegmentCache(store, version))
		require.NoError(t, err)
		assert.Equal(t, 3, completer.count())
		assert.Equal(t, "譯文1", result[0].Text)
	})

	t.Run("write failure → run still succeeds", func(t *testing.T) {
		store := newMemoryStore()
		store.setErr = errors.New("disk full")
		completer := &countingCompleter{}
		svc := NewTranslationService(completer, nil)
		result, _, outcome, err := svc.TranslateWithGlossaryHarvest(context.Background(), blocks, nil, nil,
			WithSegmentCache(store, version))
		require.NoError(t, err)
		assert.Equal(t, 3, completer.count())
		assert.Equal(t, "譯文30", result[29].Text)
		assert.Equal(t, 0, outcome.EnglishKeptBlocks)
	})
}

// Changing what the translation depends on must re-translate, not serve the
// old rendering back. That is the whole point of keying on RunVersion.
func TestTranslate_ChangedRunVersionMissesEverything(t *testing.T) {
	blocks := cacheTestBlocks(10)
	store := newMemoryStore()
	base := testRunVersion()

	seed := &countingCompleter{}
	require.NotNil(t, NewTranslationService(seed, nil))
	_, _, _, err := NewTranslationService(seed, nil).TranslateWithGlossaryHarvest(
		context.Background(), blocks, nil, nil, WithSegmentCache(store, base))
	require.NoError(t, err)
	require.Equal(t, 1, seed.count())

	for name, changed := range map[string]models.RunVersion{
		"another model":      {MetadataHash: "m", GlossaryVersion: "g", PromptVersion: "p", ModelID: "claude-haiku-4-5"},
		"a grown glossary":   {MetadataHash: "m", GlossaryVersion: "g2", PromptVersion: "p", ModelID: "claude-sonnet-5"},
		"a bumped prompt":    {MetadataHash: "m", GlossaryVersion: "g", PromptVersion: "p2", ModelID: "claude-sonnet-5"},
		"refreshed metadata": {MetadataHash: "m2", GlossaryVersion: "g", PromptVersion: "p", ModelID: "claude-sonnet-5"},
	} {
		c := &countingCompleter{}
		_, _, _, err := NewTranslationService(c, nil).TranslateWithGlossaryHarvest(
			context.Background(), blocks, nil, nil, WithSegmentCache(store, changed))
		require.NoError(t, err, name)
		assert.Equalf(t, 1, c.count(), "%s → every cue is a miss", name)
	}

	same := &countingCompleter{}
	_, _, _, err = NewTranslationService(same, nil).TranslateWithGlossaryHarvest(
		context.Background(), blocks, nil, nil, WithSegmentCache(store, base))
	require.NoError(t, err)
	assert.Equal(t, 0, same.count())
}

// A cue the model failed to return keeps English. Storing that would poison the
// cache: every later run would serve the English back as if translated.
func TestTranslate_EnglishKeptCuesAreNeverStored(t *testing.T) {
	blocks := cacheTestBlocks(10)
	store := newMemoryStore()
	// Answers only 8 of the 10 indices.
	partial := &partialCompleter{skip: map[int]bool{3: true, 7: true}}
	svc := NewTranslationService(partial, nil)

	result, _, outcome, err := svc.TranslateWithGlossaryHarvest(context.Background(), blocks, nil, nil,
		WithSegmentCache(store, testRunVersion()))

	require.NoError(t, err)
	assert.Equal(t, 2, outcome.EnglishKeptBlocks)
	assert.Equal(t, "Line 3", result[2].Text)
	assert.Equal(t, 8, store.size(), "the two English cues stayed out of the cache")
	assert.NotContains(t, store.entries, segkey.SegmentKey(blocks[2].Text, testRunVersion()))
}

type partialCompleter struct {
	skip  map[int]bool
	calls int
}

func (p *partialCompleter) CompleteText(_ context.Context, _, userPrompt string, _ int) (string, error) {
	p.calls++
	var out strings.Builder
	for _, idx := range promptIndices(userPrompt) {
		if p.skip[idx] {
			continue
		}
		fmt.Fprintf(&out, "[%d] 譯文%d\n", idx, idx)
	}
	return out.String(), nil
}

// The pre-story path must stay byte-identical for every caller that passes no
// store — the .nfo metadata localization leg included.
func TestTranslate_NoStoreIsTheOldBehaviour(t *testing.T) {
	blocks := cacheTestBlocks(25)

	withNil := &countingCompleter{}
	a, _, outA, errA := NewTranslationService(withNil, nil).TranslateWithGlossaryHarvest(
		context.Background(), blocks, nil, nil)
	require.NoError(t, errA)

	plain := &countingCompleter{}
	b, _, outB, errB := NewTranslationService(plain, nil).TranslateWithGlossaryHarvest(
		context.Background(), blocks, nil, nil, WithSegmentCache(nil, testRunVersion()))
	require.NoError(t, errB)

	assert.Equal(t, a, b)
	assert.Equal(t, outA, outB)
	assert.Equal(t, withNil.count(), plain.count())
	assert.Equal(t, 3, withNil.count())
}

// A cancelled run keeps exactly what it paid for: the batches that came back
// before the cancel are in the store, and the next run starts from there.
func TestTranslate_CancelledRunKeepsWhatItPaidFor(t *testing.T) {
	blocks := cacheTestBlocks(50)
	store := newMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	// Cancelling from INSIDE the third call makes the race deterministic —
	// a timer would finish after a 5-batch run that takes microseconds.
	completer := &cancellingCompleter{cancelAt: 3, cancel: cancel}
	svc := NewTranslationService(completer, nil)

	_, _, _, err := svc.TranslateWithGlossaryHarvest(ctx, blocks, nil, nil, WithSegmentCache(store, testRunVersion()))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
	assert.Equal(t, 20, store.size(), "the two batches that completed before the cancel are kept, nothing more")

	// …and the next run picks up exactly there.
	resumed := &countingCompleter{}
	_, _, _, err = NewTranslationService(resumed, nil).TranslateWithGlossaryHarvest(
		context.Background(), blocks, nil, nil, WithSegmentCache(store, testRunVersion()))
	require.NoError(t, err)
	assert.Equal(t, 3, resumed.count(), "30 remaining cues = 3 batches")
}

type cancellingCompleter struct {
	countingCompleter
	cancelAt int
	cancel   context.CancelFunc
}

func (c *cancellingCompleter) CompleteText(ctx context.Context, sys, userPrompt string, maxTokens int) (string, error) {
	c.mu.Lock()
	nth := c.calls + 1
	c.mu.Unlock()
	if nth == c.cancelAt {
		c.cancel()
		c.mu.Lock()
		c.calls++
		c.mu.Unlock()
		return "", context.Canceled
	}
	return c.countingCompleter.CompleteText(ctx, sys, userPrompt, maxTokens)
}

// The prompt's media context and glossary still reach the model on the resumed
// run — the cache changes WHICH cues are sent, never HOW they are asked.
func TestTranslate_ResumeKeepsPromptShape(t *testing.T) {
	blocks := cacheTestBlocks(20)
	store := newMemoryStore()
	version := testRunVersion()
	for i := 0; i < 10; i++ {
		require.NoError(t, store.Set(context.Background(),
			segkey.SegmentKey(blocks[i].Text, version), "舊譯文", segkey.TTL))
	}
	completer := &countingCompleter{}
	svc := NewTranslationService(completer, nil)

	_, _, _, err := svc.TranslateWithGlossaryHarvest(context.Background(), blocks,
		[]GlossaryPair{{Source: "Dumbledore", Target: "鄧不利多"}}, nil,
		WithSegmentCache(store, version),
		WithMediaMetadata(prompts.MediaMetadata{Title: "Goblet of Fire", Year: 2005}))

	require.NoError(t, err)
	require.Equal(t, 1, completer.count())
	assert.Contains(t, completer.prompts[0], "鄧不利多", "the glossary still rides every batch")
}
