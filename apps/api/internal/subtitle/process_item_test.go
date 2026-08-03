package subtitle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// ─── Item-flow fakes ───────────────────────────────────────────────────────

// spyRouter answers with a canned verdict and counts probes, so "the pre-flight
// spent nothing" is provable rather than asserted.
type spyRouter struct {
	decision RouteDecision
	err      error
	calls    int
	order    *[]string
}

func (r *spyRouter) SelectAndRoute(_ context.Context, _, _ string) (RouteDecision, error) {
	r.calls++
	if r.order != nil {
		*r.order = append(*r.order, "route")
	}
	if r.err != nil {
		return RouteDecision{}, r.err
	}
	return r.decision, nil
}

// recordingPlacer captures the exact bytes and request handed to the sole
// sidecar writer.
type recordingPlacer struct {
	requests []PlaceRequest
	path     string
	err      error
	order    *[]string
}

func (p *recordingPlacer) Place(req PlaceRequest) (*PlaceResult, error) {
	p.requests = append(p.requests, req)
	if p.order != nil {
		*p.order = append(*p.order, "place")
	}
	if p.err != nil {
		return nil, p.err
	}
	return &PlaceResult{SubtitlePath: p.path, Language: req.Language}, nil
}

type statusWrite struct {
	status   models.SubtitleStatus
	path     string
	language string
}

// fakeMediaStore resolves one item and records every status transition in order.
type fakeMediaStore struct {
	item     *MediaItem
	loadErr  error
	writes   []statusWrite
	writeErr error
	order    *[]string
}

func (s *fakeMediaStore) Load(_ context.Context, _ MediaRef) (*MediaItem, error) {
	if s.loadErr != nil {
		return nil, s.loadErr
	}
	return s.item, nil
}

func (s *fakeMediaStore) SetSubtitleStatus(_ context.Context, _ MediaRef, status models.SubtitleStatus, path, language string) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	s.writes = append(s.writes, statusWrite{status: status, path: path, language: language})
	if s.order != nil {
		*s.order = append(*s.order, "media:"+string(status))
	}
	return nil
}

func (s *fakeMediaStore) statuses() []models.SubtitleStatus {
	out := make([]models.SubtitleStatus, 0, len(s.writes))
	for _, w := range s.writes {
		out = append(out, w.status)
	}
	return out
}

// ─── Harness ───────────────────────────────────────────────────────────────

type itemHarness struct {
	pipeline  *Pipeline
	ref       MediaRef
	mediaPath string
	router    *spyRouter
	placer    *recordingPlacer
	media     *fakeMediaStore
	runs      *fakeRunStore
	cache     *memorySegmentCache
	trans     *fakeTranslator
	conv      *recordingConverter
	order     *[]string
	progress  []string
}

// newItemHarness wires a fully-ported pipeline over fakes, with a translator
// that echoes a Traditional-Chinese line for every requested cue.
func newItemHarness(t *testing.T, decision RouteDecision, opts ...PipelineOption) *itemHarness {
	t.Helper()

	order := &[]string{}
	mediaPath := newMediaFile(t)

	h := &itemHarness{
		ref:       MediaRef{ID: "ep-1", MediaType: models.SubtitleRunMediaEpisode},
		mediaPath: mediaPath,
		router:    &spyRouter{decision: decision, order: order},
		placer:    &recordingPlacer{path: ExpectedSidecarPath(mediaPath), order: order},
		runs:      &fakeRunStore{order: order},
		cache:     newMemorySegmentCache(),
		conv:      &recordingConverter{},
		order:     order,
	}
	h.media = &fakeMediaStore{
		item: &MediaItem{
			FilePath: mediaPath,
			TMDbID:   func() *int64 { v := int64(1399); return &v }(),
			ShowKey:  "series-42",
			Context:  richContext(),
		},
		order: order,
	}
	h.trans = &fakeTranslator{
		fn: func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			out := make(map[int]string, len(blocks))
			for _, b := range blocks {
				out[b.Index] = "早安"
			}
			return out, ai.CompletionUsage{CacheCreationInputTokens: 5000}, nil
		},
	}

	base := []PipelineOption{
		WithRouter(h.router),
		WithPlacer(h.placer),
		WithMediaStore(h.media),
		WithRunStore(h.runs),
		WithSegmentCache(h.cache),
		WithModelID("claude-haiku-4-5"),
		WithProgress(func(_ MediaRef, stage PipelineStage, _ string) {
			h.progress = append(h.progress, string(stage))
		}),
	}
	h.pipeline = NewPipeline(h.trans, h.conv, nil, append(base, opts...)...)
	return h
}

func englishTrack(texts ...string) *ExtractedTrack {
	return &ExtractedTrack{StreamIndex: 2, Language: "eng", Codec: "subrip", Blocks: cues(texts...)}
}

func translateDecision(texts ...string) RouteDecision {
	return RouteDecision{
		Kind:            RouteTranslate,
		Track:           englishTrack(texts...),
		DetectedVariant: LangUndetermined,
		Reason:          "stream 2 tagged eng carries no CJK content — queue for translation",
	}
}

// ─── AC #1: verdict branches ───────────────────────────────────────────────

func TestProcessItem_VerdictBranches(t *testing.T) {
	tests := []struct {
		name        string
		decision    RouteDecision
		wantRun     models.SubtitleRunStatus
		wantStatus  []models.SubtitleStatus
		wantPlaced  bool
		wantPayload string
		wantSource  string
	}{
		{
			name: "no usable text source",
			decision: RouteDecision{Kind: RouteNoTextSource,
				Reason: "no usable embedded text subtitle track (2 image-codec track(s), 0 text)"},
			wantRun:    models.SubtitleRunSkipped,
			wantStatus: []models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusNoTextSource},
		},
		{
			name: "deliberate skip",
			decision: RouteDecision{Kind: RouteSkip,
				Reason: "1 embedded text track(s) present but none tagged eng/en"},
			wantRun:    models.SubtitleRunSkipped,
			wantStatus: []models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusSkipped},
		},
		{
			name: "already Traditional — deliver as-is",
			decision: RouteDecision{
				Kind:            RouteDeliverDirect,
				Track:           &ExtractedTrack{StreamIndex: 3, Language: "eng", Blocks: cues("早安", "待會見")},
				DetectedVariant: LangTraditional,
			},
			wantRun:     models.SubtitleRunCompleted,
			wantStatus:  []models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusFound},
			wantPlaced:  true,
			wantPayload: "早安",
			wantSource:  LangTraditional,
		},
		{
			name: "Simplified — convert then deliver",
			decision: RouteDecision{
				Kind:            RouteConvertThenDeliver,
				Track:           &ExtractedTrack{StreamIndex: 3, Language: "eng", Blocks: cues("这个软件很好用")},
				DetectedVariant: LangSimplified,
			},
			wantRun:     models.SubtitleRunCompleted,
			wantStatus:  []models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusFound},
			wantPlaced:  true,
			wantPayload: "這個軟件很好用", // s2twpFake converts 这→這 个→個 软→軟
			wantSource:  LangSimplified,
		},
		{
			name:       "English — translate",
			decision:   translateDecision("Good morning.", "See you later."),
			wantRun:    models.SubtitleRunCompleted,
			wantStatus: []models.SubtitleStatus{models.SubtitleStatusExtracting, models.SubtitleStatusTranslating, models.SubtitleStatusFound},
			wantPlaced: true,
			wantSource: "eng",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newItemHarness(t, tc.decision)

			outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
			require.NoError(t, err)
			require.NotNil(t, outcome)

			assert.Equal(t, tc.decision.Kind, outcome.Kind)
			assert.Equal(t, tc.wantStatus, h.media.statuses())

			require.Len(t, h.runs.created, 1, "exactly one run row per item")
			assert.Equal(t, models.SubtitleRunPending, h.runs.created[0].Status, "the row is born pending")
			final := h.runs.lastUpdate(t)
			assert.Equal(t, tc.wantRun, final.Status)
			assert.NotNil(t, final.CompletedAt, "a terminal run must stamp completed_at")

			if !tc.wantPlaced {
				assert.Empty(t, h.placer.requests, "a routed-out item must never touch the media folder")
				assert.Equal(t, tc.decision.Reason, final.ErrorMessage, "the routing reason is the skip's record")
				return
			}

			require.Len(t, h.placer.requests, 1)
			req := h.placer.requests[0]
			assert.Equal(t, h.mediaPath, req.MediaFilePath)
			assert.Equal(t, "zh-Hant", req.Language)
			assert.Equal(t, "srt", req.Format)
			assert.Zero(t, req.Score, "a generated subtitle has no search score (AC #6.3)")
			if tc.wantPayload != "" {
				assert.Contains(t, string(req.SubtitleData), tc.wantPayload)
			}

			assert.Equal(t, h.placer.path, outcome.SubtitlePath)
			assert.Equal(t, h.placer.path, final.OutputPath)
			assert.Equal(t, len(tc.decision.Track.Blocks), final.CueCount)
			assert.Equal(t, tc.wantSource, final.SourceLanguage)
			assert.Equal(t, models.SubtitleStatusFound, h.media.writes[len(h.media.writes)-1].status)
			assert.Equal(t, "zh-Hant", h.media.writes[len(h.media.writes)-1].language)
		})
	}
}

// TestProcessItem_RecordsTheVersionTuple — provenance is only useful if it
// records WHICH inputs produced the file (the M1 pilot attributes results with
// exactly these four columns).
func TestProcessItem_RecordsTheVersionTuple(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	final := h.runs.lastUpdate(t)
	assert.Equal(t, MetadataHash(richContext()), final.MetadataHash)
	assert.Empty(t, final.GlossaryVersion)
	assert.Equal(t, prompts.SubtitleTranslatorPromptVersion, final.PromptVersion)
	assert.Equal(t, "claude-haiku-4-5", final.ModelID)
	require.NotNil(t, final.TMDbID)
	assert.Equal(t, int64(1399), *final.TMDbID)
	assert.True(t, final.CacheEnabled, "the fake reports cache-creation tokens on the first chunk")
}

// TestProcessItem_ProvenanceIsWrittenStrictlyAfterThePlace is P9. The reverse
// order leaves a `completed` row claiming a file a crash never wrote.
func TestProcessItem_ProvenanceIsWrittenStrictlyAfterThePlace(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	order := *h.order
	place := indexOfLast(order, "place")
	completed := indexOfLast(order, "run:completed")
	found := indexOfLast(order, "media:found")

	require.NotEqual(t, -1, place)
	require.NotEqual(t, -1, completed)
	require.NotEqual(t, -1, found)
	assert.Less(t, place, completed, "provenance must be written AFTER the file lands (P9)")
	assert.Less(t, completed, found, "the media row goes terminal last")
}

// ─── AC #2: pre-flight spends nothing ──────────────────────────────────────

func TestProcessItem_PreflightEarlyExitSpendsNothing(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	require.NoError(t, os.WriteFile(ExpectedSidecarPath(h.mediaPath), []byte(oneCueSRT), 0o600))

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	require.NotNil(t, outcome)
	assert.Nil(t, outcome.Run, "an early-exit writes no provenance row")
	assert.Equal(t, ExpectedSidecarPath(h.mediaPath), outcome.SubtitlePath)

	assert.Zero(t, h.router.calls, "no ffprobe/ffmpeg may run before the P5 predicate has spoken")
	assert.Empty(t, h.trans.calls, "no LLM call may run either")
	assert.Empty(t, h.runs.created)
	assert.Empty(t, h.media.writes, "the media row must not churn through extracting for a no-op")
	assert.Empty(t, h.placer.requests)
}

func TestProcessItem_ForceBypassesPreflightAndCacheReadsButStillWrites(t *testing.T) {
	source := cues("Good morning.")
	h := newItemHarness(t, RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: source},
		DetectedVariant: LangUndetermined,
	})
	require.NoError(t, os.WriteFile(ExpectedSidecarPath(h.mediaPath), []byte(oneCueSRT), 0o600))

	version := h.pipeline.runVersion(richContext())
	h.cache.entries[segmentKey(source[0].Text, version)] = "早安（舊快取）"

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{Force: true})
	require.NoError(t, err)

	require.NotNil(t, outcome.Run, "force runs the item for real, so provenance IS written")
	assert.Equal(t, 1, h.router.calls, "force bypasses the P5 early-exit")
	assert.Empty(t, h.cache.reads, "force bypasses segment-cache READS")
	require.Len(t, h.trans.calls, 1, "the cached cue must be re-translated")

	assert.Equal(t, "早安", h.cache.writes[segmentKey(source[0].Text, version)],
		"force still WRITES — the refreshed translation replaces the stale entry")
	assert.Contains(t, string(h.placer.requests[0].SubtitleData), "早安")
	assert.NotContains(t, string(h.placer.requests[0].SubtitleData), "舊快取")
}

// ─── AC #3: cache hits skip the LLM ────────────────────────────────────────

func TestProcessItem_CacheHitsNeverReachTheTranslator(t *testing.T) {
	source := cues("Good morning.", "This software is great.", "See you later.")
	h := newItemHarness(t, RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: source},
		DetectedVariant: LangUndetermined,
	})

	version := h.pipeline.runVersion(richContext())
	h.cache.entries[segmentKey(source[1].Text, version)] = "這個軟體很好用"

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	require.Len(t, h.trans.calls, 1)
	assert.Equal(t, []int{1, 3}, h.trans.calls[0].indexes(), "only the misses may be sent")

	payload := string(h.placer.requests[0].SubtitleData)
	assert.Contains(t, payload, "這個軟體很好用", "the cached cue must still be delivered")

	blocks, err := ParseSRT(payload)
	require.NoError(t, err)
	require.Len(t, blocks, 3, "the merged track carries every source cue")
	for i, b := range blocks {
		assert.Equal(t, source[i].Index, b.Index)
		assert.Equal(t, source[i].Start, b.Start)
		assert.Equal(t, source[i].End, b.End)
	}
}

func TestProcessItem_FullCacheHitSkipsTheLLMEntirely(t *testing.T) {
	source := cues("Good morning.", "See you later.")
	h := newItemHarness(t, RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: source},
		DetectedVariant: LangUndetermined,
	})

	version := h.pipeline.runVersion(richContext())
	for _, b := range source {
		h.cache.entries[segmentKey(b.Text, version)] = "快取譯文"
	}

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	assert.Empty(t, h.trans.calls, "a fully cached item costs zero tokens")
	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunCompleted, final.Status)
	assert.False(t, final.CacheEnabled,
		"no request was sent, so no prompt prefix was cached — the column describes the PROMPT cache, not this one")
}

// echoingTranslator answers every cue with a Traditional line except the given
// source texts, which it echoes back — an echo fails the quality gate on every
// retry and ends up stubborn (NFR-R1 fail-soft: it ships its English original).
func echoingTranslator(stubborn ...string) func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
	echo := make(map[string]struct{}, len(stubborn))
	for _, text := range stubborn {
		echo[text] = struct{}{}
	}
	return func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		out := make(map[int]string, len(blocks))
		for _, b := range blocks {
			if _, bad := echo[b.Text]; bad {
				out[b.Index] = b.Text
				continue
			}
			out[b.Index] = "早安"
		}
		return out, ai.CompletionUsage{CacheCreationInputTokens: 5000}, nil
	}
}

func longTrack(n int) []string {
	texts := make([]string, n)
	for i := range texts {
		texts[i] = fmt.Sprintf("Dialogue line %d here.", i)
	}
	return texts
}

// TestProcessItem_StubbornCeilingIsMeasuredAgainstTheDeliveredTrack — FR16's 5%
// ceiling is a property of what SHIPS, not of whatever subset reached the LLM.
// Cache hits are already-accepted translations, so they belong in the
// denominator: without this, the ceiling silently tightens as the cache warms
// and one flaky cue kills a 95%-cached, fully-deliverable episode.
func TestProcessItem_StubbornCeilingIsMeasuredAgainstTheDeliveredTrack(t *testing.T) {
	texts := longTrack(20)
	source := cues(texts...)
	h := newItemHarness(t, RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: source},
		DetectedVariant: LangUndetermined,
	})

	// 18 of 20 cues are warm; 2 reach the LLM and one of those is stubborn.
	version := h.pipeline.runVersion(richContext())
	for i := 0; i < 18; i++ {
		h.cache.entries[segmentKey(source[i].Text, version)] = "快取譯文"
	}
	h.trans.fn = echoingTranslator(texts[19])

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})

	require.NoError(t, err,
		"1 stubborn cue in a 20-cue track is exactly the 5%% ceiling — a warm cache must not turn it into 1-of-2")
	require.Len(t, h.placer.requests, 1)
	blocks, err := ParseSRT(string(h.placer.requests[0].SubtitleData))
	require.NoError(t, err)
	assert.Len(t, blocks, 20, "the whole episode still ships")
}

// TestProcessItem_TheCeilingStillFiresOnAGenuinelyBrokenTrack — the fix must
// widen the denominator, not disable the guard.
func TestProcessItem_TheCeilingStillFiresOnAGenuinelyBrokenTrack(t *testing.T) {
	texts := longTrack(20)
	h := newItemHarness(t, translateDecision(texts...))
	h.trans.fn = echoingTranslator(texts[0], texts[1], texts[2]) // 3 of 20 = 15%

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})

	require.Error(t, err, "a track shipping 15%% English must still fail the item")
	assert.Contains(t, err.Error(), "ceiling")
	assert.Empty(t, h.placer.requests, "nothing half-English may reach the media folder")
}

// TestProcessItem_StubbornCuesAreNeverCached — a stubborn cue ships its ENGLISH
// original. Caching that would freeze a transient failure for the full 30-day
// TTL, and because the key is cue content + show metadata, the same English
// line would then be served to every other episode of the show with no retry
// ever firing again short of --force or a version bump.
func TestProcessItem_StubbornCuesAreNeverCached(t *testing.T) {
	texts := longTrack(40)
	source := cues(texts...)
	h := newItemHarness(t, RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: source},
		DetectedVariant: LangUndetermined,
	})
	h.trans.fn = echoingTranslator(texts[1]) // 1 of 40 — under the ceiling, so the item ships

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	version := h.pipeline.runVersion(richContext())
	_, cached := h.cache.writes[segmentKey(source[1].Text, version)]
	assert.False(t, cached, "the English fail-soft fallback must not be persisted as if it were a translation")

	// Its neighbours, which really were translated, ARE cached.
	assert.Equal(t, "早安", h.cache.writes[segmentKey(source[0].Text, version)])
	assert.Equal(t, "早安", h.cache.writes[segmentKey(source[2].Text, version)])

	// And the cue still SHIPS with its English original — fail-soft is intact,
	// it is only the persistence that was wrong.
	assert.Contains(t, string(h.placer.requests[0].SubtitleData), texts[1])
}

// TestProcessItem_RequestsSentDisambiguatesTheCacheVerdict — cache_enabled is a
// non-nullable bool, so a fully segment-cached item (which sends nothing) and an
// item whose prompt prefix silently failed to cache both land on `false`. The
// scope's request counter is what lets the M1 pilot tell them apart.
func TestProcessItem_RequestsSentDisambiguatesTheCacheVerdict(t *testing.T) {
	source := cues("Good morning.", "See you later.")
	decision := RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: source},
		DetectedVariant: LangUndetermined,
	}

	t.Run("inert prefix — sent, but nothing cached", func(t *testing.T) {
		h := newItemHarness(t, decision)
		h.trans.fn = func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			out := make(map[int]string, len(blocks))
			for _, b := range blocks {
				out[b.Index] = "早安"
			}
			return out, ai.CompletionUsage{InputTokens: 900}, nil // both cache fields zero
		}

		_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
		require.NoError(t, err)
		assert.False(t, h.runs.lastUpdate(t).CacheEnabled)
		assert.Len(t, h.trans.calls, 1, "a request WAS issued — cache_enabled=false is the AC #4.2 verdict")
	})

	t.Run("fully cached — nothing sent at all", func(t *testing.T) {
		h := newItemHarness(t, decision)
		version := h.pipeline.runVersion(richContext())
		for _, b := range source {
			h.cache.entries[segmentKey(b.Text, version)] = "快取譯文"
		}

		_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
		require.NoError(t, err)
		assert.False(t, h.runs.lastUpdate(t).CacheEnabled)
		assert.Empty(t, h.trans.calls, "no request was issued — the SAME column value, a different cause")
	})
}

// ─── AC #1.5: failure policy ───────────────────────────────────────────────

func TestProcessItem_FailurePolicy(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(*itemHarness)
	}{
		{"router failure", func(h *itemHarness) { h.router.err = errors.New("ffprobe exploded") }},
		{"translator failure", func(h *itemHarness) {
			h.trans.fn = func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
				return nil, ai.CompletionUsage{}, errors.New("provider 500")
			}
		}},
		{"placer failure", func(h *itemHarness) { h.placer.err = errors.New("read-only filesystem") }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newItemHarness(t, translateDecision("Good morning."))
			tc.arrange(h)

			outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
			require.Error(t, err)
			assert.Nil(t, outcome)

			final := h.runs.lastUpdate(t)
			assert.Equal(t, models.SubtitleRunFailed, final.Status)
			assert.NotEmpty(t, final.ErrorMessage, "the run row is the audit trail")
			assert.NotNil(t, final.CompletedAt)

			last := h.media.writes[len(h.media.writes)-1]
			assert.Equal(t, models.SubtitleStatusNotSearched, last.status,
				"the item must stay retryable — SubtitleStatus has no `failed`, and leaving it `translating` "+
					"would strand a phantom in-flight state across a restart")
			assert.Empty(t, last.path)
		})
	}
}

func TestProcessItem_FailureMessageIsBounded(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	h.router.err = errors.New(strings.Repeat("ffmpeg stderr ", 500))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.Error(t, err)

	msg := h.runs.lastUpdate(t).ErrorMessage
	assert.LessOrEqual(t, len(msg), maxErrorMessage+len("…"))
	assert.True(t, utf8ValidForTest(msg), "truncation must not leave a broken UTF-8 tail")
}

func utf8ValidForTest(s string) bool {
	for _, r := range s {
		if r == '�' {
			return false
		}
	}
	return true
}

// TestProcessItem_CancellationStillRecordsTheFailure — a shutdown must not
// strand items at `translating` with no run row explaining why.
func TestProcessItem_CancellationStillRecordsTheFailure(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	ctx, cancel := context.WithCancel(context.Background())
	h.trans.fn = func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		cancel()
		return nil, ai.CompletionUsage{}, context.Canceled
	}

	_, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{})
	require.Error(t, err)

	assert.Equal(t, models.SubtitleRunFailed, h.runs.lastUpdate(t).Status)
	assert.Equal(t, models.SubtitleStatusNotSearched, h.media.writes[len(h.media.writes)-1].status)
}

// ─── AC #1.6 + #8: progress hook ───────────────────────────────────────────

func TestProcessItem_ProgressHookFiresAtStageTransitionsAndOncePerChunk(t *testing.T) {
	texts := make([]string, 25) // 3 chunks at batch size 10
	for i := range texts {
		texts[i] = "Line."
	}
	h := newItemHarness(t, translateDecision(texts...))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	assert.Equal(t, []string{
		string(StageProbing), string(StageExtracting),
		string(StageTranslating), string(StageTranslating), string(StageTranslating),
		string(StagePlacing), string(StageComplete),
	}, h.progress, "once per chunk (P8's throttle grain), never per cue")
}

func TestProcessItem_ProgressHookIsOptional(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."), WithProgress(nil))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err, "a nil hook must be a no-op — SSE is sub-1-6's, not this story's")
	assert.Empty(t, h.progress)
}

// ─── Wiring guards ─────────────────────────────────────────────────────────

func TestProcessItem_UnwiredPortsFailByName(t *testing.T) {
	p := NewPipeline(&fakeTranslator{}, &recordingConverter{}, nil)

	_, err := p.ProcessItem(context.Background(), MediaRef{ID: "m1", MediaType: models.SubtitleRunMediaMovie}, ProcessItemOptions{})
	require.Error(t, err)
	for _, port := range []string{"MediaStore", "RunStore", "TrackRouter", "SubtitlePlacer", "SegmentCache"} {
		assert.Contains(t, err.Error(), port, "an unwired pipeline must name what is missing, not nil-panic mid-run")
	}
}

func TestProcessItem_MissingMediaFileIsRejectedBeforeAnyWrite(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	h.media.item = &MediaItem{}

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.Error(t, err)
	assert.Empty(t, h.runs.created)
	assert.Empty(t, h.media.writes)
}

func TestProcessItem_RejectsAVerdictWithNoCues(t *testing.T) {
	h := newItemHarness(t, RouteDecision{Kind: RouteDeliverDirect, Track: &ExtractedTrack{StreamIndex: 2}})

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleExtractFailed)
	assert.Equal(t, models.SubtitleRunFailed, h.runs.lastUpdate(t).Status)
	assert.Empty(t, h.placer.requests)
}

func TestProcessItem_ConverterFailureOnTheConvertRouteFailsTheItem(t *testing.T) {
	h := newItemHarness(t, RouteDecision{
		Kind:            RouteConvertThenDeliver,
		Track:           &ExtractedTrack{StreamIndex: 3, Language: "eng", Blocks: cues("这个软件很好用")},
		DetectedVariant: LangSimplified,
	})
	h.conv.err = errors.New("opencc unavailable")

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.Error(t, err,
		"on this route conversion IS the deliverable — shipping Simplified text as .zh-Hant.srt would be a lie")
	assert.Empty(t, h.placer.requests)
	assert.Equal(t, models.SubtitleRunFailed, h.runs.lastUpdate(t).Status)
}

// TestProcessItem_MoviesBypassTheShowGate — a movie carries no ShowKey, so two
// movies must never queue behind one another.
func TestProcessItem_MoviesBypassTheShowGate(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	h.ref = MediaRef{ID: "movie-1", MediaType: models.SubtitleRunMediaMovie}
	h.media.item.ShowKey = ""

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)
	assert.Zero(t, h.pipeline.gate.size(), "an empty show key must not even create a latch entry")
}

// ─── AC #7.5: integration over the REAL repositories ───────────────────────

// TestProcessItem_IntegrationWritesAllSixteenRunColumns runs the whole item
// flow against a real :memory: SQLite with the real SubtitleRunRepository and
// CacheRepository. Rule 15 / bugfix-20-1: a mocked repo cannot catch a column
// that never reaches the INSERT or never comes back from the SELECT.
func TestProcessItem_IntegrationWritesAllSixteenRunColumns(t *testing.T) {
	db := newMigratedTestDB(t)
	runRepo := repository.NewSubtitleRunRepository(db)
	cache := NewSegmentCacheRepository(repository.NewCacheRepository(db))

	mediaDir := t.TempDir()
	mediaPath := filepath.Join(mediaDir, "Show.S01E02.1080p.mkv")
	require.NoError(t, os.WriteFile(mediaPath, []byte("video"), 0o600))

	tmdbID := int64(1399)
	media := &fakeMediaStore{item: &MediaItem{
		FilePath: mediaPath,
		TMDbID:   &tmdbID,
		ShowKey:  "series-42",
		Context:  richContext(),
	}}

	source := cues("Good morning.", "See you later.")
	router := &spyRouter{decision: RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Codec: "subrip", Blocks: source},
		DetectedVariant: LangUndetermined,
	}}

	translator := &fakeTranslator{
		fn: func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			out := make(map[int]string, len(blocks))
			for _, b := range blocks {
				out[b.Index] = "早安"
			}
			return out, ai.CompletionUsage{CacheReadInputTokens: 4096}, nil
		},
	}

	p := NewPipeline(translator, &recordingConverter{}, nil,
		WithRouter(router),
		WithPlacer(NewPlacer(DefaultPlacerConfig())), // the REAL sole writer
		WithMediaStore(media),
		WithRunStore(runRepo),
		WithSegmentCache(cache),
		WithModelID("claude-haiku-4-5"),
	)

	ref := MediaRef{ID: "ep-77", MediaType: models.SubtitleRunMediaEpisode}
	outcome, err := p.ProcessItem(context.Background(), ref, ProcessItemOptions{})
	require.NoError(t, err)
	require.NotNil(t, outcome.Run)

	// The sidecar really landed beside the media file.
	written, err := os.ReadFile(ExpectedSidecarPath(mediaPath))
	require.NoError(t, err)
	assert.Contains(t, string(written), "早安")
	assert.Equal(t, ExpectedSidecarPath(mediaPath), outcome.SubtitlePath)

	// Every column survives INSERT → UPDATE → SELECT → Scan.
	stored, err := runRepo.FindByID(context.Background(), outcome.Run.ID)
	require.NoError(t, err)
	assert.Equal(t, ref.ID, stored.MediaID)
	assert.Equal(t, ref.MediaType, stored.MediaType)
	require.NotNil(t, stored.TMDbID)
	assert.Equal(t, tmdbID, *stored.TMDbID)
	assert.Equal(t, MetadataHash(richContext()), stored.MetadataHash)
	assert.Empty(t, stored.GlossaryVersion)
	assert.Equal(t, prompts.SubtitleTranslatorPromptVersion, stored.PromptVersion)
	assert.Equal(t, "claude-haiku-4-5", stored.ModelID)
	assert.Equal(t, models.SubtitleRunCompleted, stored.Status)
	assert.Equal(t, "eng", stored.SourceLanguage)
	assert.Equal(t, ExpectedSidecarPath(mediaPath), stored.OutputPath)
	assert.Equal(t, len(source), stored.CueCount)
	assert.True(t, stored.CacheEnabled)
	assert.Empty(t, stored.ErrorMessage)
	assert.False(t, stored.StartedAt.IsZero())
	require.NotNil(t, stored.CompletedAt)

	// The resume predicate now matches — which is what makes a second scan free.
	resumed, err := runRepo.FindCompletedRun(context.Background(), ref.ID, ref.MediaType, stored.Version())
	require.NoError(t, err)
	require.NotNil(t, resumed)
	assert.Equal(t, stored.ID, resumed.ID)

	// And the translated cues really are in cache_entries under the versioned key.
	version := p.runVersion(richContext())
	key := segmentKey(source[0].Text, version)
	values, err := cache.GetMany(context.Background(), []string{key})
	require.NoError(t, err)
	require.Contains(t, values, key, "the segment cache must be populated for the next run")
	assert.Equal(t, "早安", values[key])

	// Second pass: the P5 pre-flight sees the sidecar it just wrote and exits
	// before spending anything.
	before := router.calls
	second, err := p.ProcessItem(context.Background(), ref, ProcessItemOptions{})
	require.NoError(t, err)
	assert.Nil(t, second.Run)
	assert.Equal(t, before, router.calls, "a re-scan of a completed item must cost nothing")
}
