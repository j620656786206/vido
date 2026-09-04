package subtitle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	// onRoute (optional) sees the ctx ProcessItem hands the router — the
	// sub-6-3 extract-wait notifier rides on it.
	onRoute func(ctx context.Context)
}

func (r *spyRouter) SelectAndRoute(ctx context.Context, _, _ string) (RouteDecision, error) {
	r.calls++
	if r.onRoute != nil {
		r.onRoute(ctx)
	}
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

// ─── sub-5-5: glossary feed (AC #5) + harvest write-back (AC #4) ───────────

// fakeGlossaryStore records the feed/harvest traffic the item flow generates.
type fakeGlossaryStore struct {
	terms       map[string]string
	lookupErr   error
	lookupKeys  []string
	insertKey   string
	inserted    map[string]string
	insertErr   error
	insertCalls int
}

func (f *fakeGlossaryStore) Lookup(_ context.Context, mediaID string) (map[string]string, error) {
	f.lookupKeys = append(f.lookupKeys, mediaID)
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	return f.terms, nil
}

func (f *fakeGlossaryStore) InsertNew(_ context.Context, mediaID string, terms map[string]string) (int, error) {
	f.insertCalls++
	f.insertKey = mediaID
	f.inserted = terms
	if f.insertErr != nil {
		return 0, f.insertErr
	}
	return len(terms), nil
}

func TestProcessItem_FeedsGlossaryIntoPromptAndRunVersion(t *testing.T) {
	store := &fakeGlossaryStore{terms: map[string]string{"Vecna": "維克那", "Demogorgon": "魔王獸"}}
	h := newItemHarness(t, translateDecision("The Demogorgon is coming"), WithGlossaryStore(store))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	// The key is the SERIES id (ShowKey), never the episode's own id — that is
	// what makes harvest order-independent across a season.
	assert.Equal(t, []string{"series-42"}, store.lookupKeys)

	// The fed pairs reached the prompt's per-show system block, sorted by
	// Source so the rendered prompt is deterministic.
	require.NotEmpty(t, h.trans.calls)
	var perShow string
	for _, b := range h.trans.calls[0].sys {
		perShow += b.Text
	}
	assert.Contains(t, perShow, "Demogorgon → 魔王獸")
	assert.Contains(t, perShow, "Vecna → 維克那")
	assert.Less(t, strings.Index(perShow, "Demogorgon"), strings.Index(perShow, "Vecna"),
		"entries render in Source order")

	// GlossaryVersion on the run row is the hash of exactly what was fed.
	wantVersion := GlossaryVersionHash([]prompts.GlossaryEntry{
		{Source: "Demogorgon", Target: "魔王獸"}, {Source: "Vecna", Target: "維克那"},
	})
	require.NotEmpty(t, wantVersion)
	require.NotEmpty(t, h.runs.created)
	assert.Equal(t, wantVersion, h.runs.created[0].GlossaryVersion)
}

func TestProcessItem_MovieGlossaryKeysOnItself(t *testing.T) {
	store := &fakeGlossaryStore{}
	h := newItemHarness(t, translateDecision("Good morning."), WithGlossaryStore(store))
	h.ref = MediaRef{ID: "movie-7", MediaType: models.SubtitleRunMediaMovie}
	h.media.item.ShowKey = "" // movies carry no ShowKey

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"movie-7"}, store.lookupKeys,
		"a movie accumulates intra-film consistency under its own id")
}

func TestProcessItem_HarvestWritesBackAfterTranslateSuccess(t *testing.T) {
	store := &fakeGlossaryStore{}
	h := newItemHarness(t, translateDecision("The Demogorgon is coming"), WithGlossaryStore(store))
	h.trans.terms = func(int) map[string]string {
		return map[string]string{"Demogorgon": "魔王獸"}
	}

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	assert.Equal(t, 1, store.insertCalls)
	assert.Equal(t, "series-42", store.insertKey, "harvest accumulates under the show key")
	assert.Equal(t, map[string]string{"Demogorgon": "魔王獸"}, store.inserted)
}

// CR sub-5-5 H2: a harvested rendering is s2twp'd before the glossary write —
// the harness converter maps 软→軟, so a Simplified trailer rendering must
// never reach the store raw.
func TestProcessItem_HarvestedTermsGetOpenCC(t *testing.T) {
	store := &fakeGlossaryStore{}
	h := newItemHarness(t, translateDecision("The Software is live"), WithGlossaryStore(store))
	h.trans.terms = func(int) map[string]string {
		return map[string]string{"The Software": "软件"}
	}

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"The Software": "軟件"}, store.inserted,
		"the rendering is converted before it becomes a MANDATORY glossary feed")
}

func TestProcessItem_FailedTranslateHarvestsNothing(t *testing.T) {
	store := &fakeGlossaryStore{}
	h := newItemHarness(t, translateDecision("Good morning."), WithGlossaryStore(store))
	h.trans.fn = func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		return nil, ai.CompletionUsage{}, errors.New("provider down")
	}
	h.trans.terms = func(int) map[string]string {
		return map[string]string{"Demogorgon": "魔王獸"}
	}

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.Error(t, err)
	assert.Zero(t, store.insertCalls,
		"a failed run ships no subtitle — feeding its terms back would only pollute (AC #4)")
}

func TestProcessItem_GlossaryLookupFailureFailsSoft(t *testing.T) {
	store := &fakeGlossaryStore{lookupErr: errors.New("db locked")}
	h := newItemHarness(t, translateDecision("Good morning."), WithGlossaryStore(store))

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err, "a glossary miss costs consistency, never the episode (Rule 13 case 3)")
	require.NotNil(t, outcome.Run)
	assert.Equal(t, models.SubtitleRunCompleted, outcome.Run.Status)
	assert.Equal(t, "", h.runs.created[0].GlossaryVersion,
		"empty feed hashes to \"\" — cache key and prompt content agree by construction")
}

func TestProcessItem_HarvestWriteFailureFailsSoft(t *testing.T) {
	store := &fakeGlossaryStore{insertErr: errors.New("disk full")}
	h := newItemHarness(t, translateDecision("Good morning."), WithGlossaryStore(store))
	h.trans.terms = func(int) map[string]string {
		return map[string]string{"Demogorgon": "魔王獸"}
	}

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err, "the translation already succeeded — a lost harvest is re-collected later, never a failed item")
	require.NotNil(t, outcome.Run)
	assert.Equal(t, models.SubtitleRunCompleted, outcome.Run.Status)
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

	version := h.pipeline.runVersion(context.Background(), richContext())
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

	version := h.pipeline.runVersion(context.Background(), richContext())
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

	version := h.pipeline.runVersion(context.Background(), richContext())
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
	version := h.pipeline.runVersion(context.Background(), richContext())
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

	version := h.pipeline.runVersion(context.Background(), richContext())
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
		version := h.pipeline.runVersion(context.Background(), richContext())
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

// ─── bugfix-autogenerator-no-timeout-or-shutdown AC #5: the cancelled marker ─

func TestProcessItem_CancellationMarksTheRunAsCancelled(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	ctx, cancel := context.WithCancel(context.Background())
	h.trans.fn = func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		cancel()
		return nil, ai.CompletionUsage{}, context.Canceled
	}

	_, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{})
	require.Error(t, err)

	last := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunFailed, last.Status)
	assert.True(t, strings.HasPrefix(last.ErrorMessage, CancelledRunPrefix),
		"a shutdown-cancelled run must carry the marker so AutoGenerator does not count it: %q", last.ErrorMessage)
}

// The ctx half of the check: ffprobe killed by cancellation surfaces as an
// *exec.ExitError, not context.Canceled — the cause alone would miss it.
func TestProcessItem_KilledProbeUnderCancellationIsMarkedCancelled(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h.router.err = errors.New("ffprobe: signal: killed")

	_, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{})
	require.Error(t, err)

	last := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunFailed, last.Status)
	assert.True(t, strings.HasPrefix(last.ErrorMessage, CancelledRunPrefix),
		"an opaque subprocess error under a cancelled ctx is still a cancellation: %q", last.ErrorMessage)
}

// A per-item deadline is about the FILE and must keep counting toward parking.
func TestProcessItem_DeadlineExceededIsNotMarkedCancelled(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	h.trans.fn = func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		return nil, ai.CompletionUsage{}, context.DeadlineExceeded
	}

	_, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{})
	require.Error(t, err)

	last := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunFailed, last.Status)
	assert.False(t, strings.HasPrefix(last.ErrorMessage, CancelledRunPrefix),
		"a timeout is a verdict on the file, not on the caller: %q", last.ErrorMessage)
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
	version := p.runVersion(context.Background(), richContext())
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

// ─── sub-5-1 AC #3: per-item budget on the FR12/pool path ──────────────────

// ctxBudgetSpy wraps the harness translator, capturing the Budget each chunk's
// ctx carries — the exact seam the real ai client reads via governed(). onChunk
// runs BEFORE the inner translator, emulating governed()'s budget pre-check.
type ctxBudgetSpy struct {
	inner   ChunkTranslator
	budgets []*ai.Budget
	onChunk func(b *ai.Budget) error
}

func (s *ctxBudgetSpy) TranslateChunk(ctx context.Context, sys []ai.SystemBlock, contextBlocks, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, map[string]string, ai.CompletionUsage, error) {
	b := ai.BudgetFromContext(ctx)
	s.budgets = append(s.budgets, b)
	if s.onChunk != nil {
		if err := s.onChunk(b); err != nil {
			return nil, nil, ai.CompletionUsage{}, err
		}
	}
	return s.inner.TranslateChunk(ctx, sys, contextBlocks, blocks)
}

func TestProcessItem_AttachesPerItemBudgetWithTheConfiguredCeiling(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."), WithRunBudgetUSD(5))
	spy := &ctxBudgetSpy{
		inner: h.trans,
		onChunk: func(b *ai.Budget) error {
			// The real claude client records usage against exactly this budget.
			b.RecordLLM("claude-haiku-4-5", 100_000, 10_000)
			return nil
		},
	}
	h.pipeline.translator = spy

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	require.NotEmpty(t, spy.budgets)
	b := spy.budgets[0]
	require.NotNil(t, b,
		"the FR12/pool path must carry a Budget — an absent one made 'AI 花費上限' a fiction for translation spend")
	snap := b.Snapshot()
	assert.InDelta(t, 5.0, snap.BudgetUSD, 1e-9, "ceiling = the wired AI_RUN_BUDGET_USD value")
	assert.Positive(t, snap.SpentUSD, "the translate leg's LLM spend is now recorded")
	assert.Equal(t, 1, snap.LLMCalls)
}

func TestProcessItem_NeverOverridesACtxBudget_ConsentCeilingRedLine(t *testing.T) {
	// sub-4-2 red line: the consent batch attaches ONE shared Budget for the
	// whole batch. The per-item envelope must NOT replace it — that would
	// silently void the ceiling the user confirmed on F16.
	h := newItemHarness(t, translateDecision("Good morning."), WithRunBudgetUSD(5))
	spy := &ctxBudgetSpy{inner: h.trans}
	h.pipeline.translator = spy

	shared := ai.NewBudget(3)
	ctx := ai.WithBudget(context.Background(), shared)
	_, err := h.pipeline.ProcessItem(ctx, h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	require.NotEmpty(t, spy.budgets)
	assert.Same(t, shared, spy.budgets[0],
		"a ctx that already carries a Budget keeps it — batch shared ceiling MUST NOT be overridden")
	assert.InDelta(t, 3.0, shared.Snapshot().BudgetUSD, 1e-9)
}

func TestProcessItem_TranslateLegBudgetCeilingFailsTheItem(t *testing.T) {
	// 11 cues → 2 chunks (batch size 10). Chunk 1 records spend over the tiny
	// ceiling; chunk 2's pre-check (the governed() emulation) short-circuits
	// with ai.ErrBudgetExceeded — the translate leg's ceiling hit is a FAIL
	// (retryable, audited), not the ASR leg's pause.
	texts := make([]string, 11)
	for i := range texts {
		texts[i] = fmt.Sprintf("Line %d.", i+1)
	}
	h := newItemHarness(t, translateDecision(texts...), WithRunBudgetUSD(0.001))
	spy := &ctxBudgetSpy{
		inner: h.trans,
		onChunk: func(b *ai.Budget) error {
			if b.Exceeded() {
				return ai.ErrBudgetExceeded
			}
			b.RecordLLM("claude-haiku-4-5", 1_000_000, 0) // $0.10 → over $0.001
			return nil
		},
	}
	h.pipeline.translator = spy

	outcome, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.Error(t, err)
	assert.Nil(t, outcome)
	assert.ErrorIs(t, err, ai.ErrBudgetExceeded, "the sentinel must survive the wrapping for caller classification")

	final := h.runs.lastUpdate(t)
	assert.Equal(t, models.SubtitleRunFailed, final.Status)
	last := h.media.writes[len(h.media.writes)-1]
	assert.Equal(t, models.SubtitleStatusNotSearched, last.status, "budget-failed items stay retryable")
}

// ─── sub-6-1: the writable pre-flight refuses BEFORE any paid call ──────────

func TestProcessItem_UnwritableTargetSpendsNothing(t *testing.T) {
	h := newItemHarness(t, RouteDecision{Kind: RouteTranslate, Track: englishTrack("Hello", "World")},
		WithWritableProbe(func(context.Context, string) error { return errors.New("read-only file system") }))
	h.media.item.SubtitleStatus = models.SubtitleStatusNotFound
	h.media.item.SubtitlePath = ""

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})

	require.ErrorIs(t, err, ErrSubtitleTargetNotWritable)
	assert.Empty(t, h.trans.calls, "no LLM call may happen against a target the placer cannot write")
	assert.Empty(t, h.placer.requests, "nothing is placed")
	// Routing (free: ffprobe + local extract) IS allowed to run — it is what
	// tells a RouteSkip apart from a writing route (CR H2).
	require.NotEmpty(t, h.runs.updated, "the run row is created pending and then updated to failed")
	run := h.runs.updated[len(h.runs.updated)-1]
	assert.Equal(t, models.SubtitleRunFailed, run.Status, "a failed run row is the audit trail (AC #3)")
	assert.Contains(t, run.ErrorMessage, "SUBTITLE_TARGET_NOT_WRITABLE")
	if run.SpentUSD != nil {
		assert.Zero(t, *run.SpentUSD, "cost must be $0")
	}
	// The media row goes back where Load found it — not_found, not not_searched.
	last := h.media.writes[len(h.media.writes)-1]
	assert.Equal(t, models.SubtitleStatusNotFound, last.status, "pre-flight refusal restores the original status (FreeOnly brake posture)")
}

func TestProcessItem_UnwritableTargetStillRecordsRouteSkip(t *testing.T) {
	// CR H2: a file the router declines writes nothing, so a read-only share
	// must not turn its free, terminal `skipped` verdict into a failure.
	h := newItemHarness(t, RouteDecision{Kind: RouteSkip, Reason: "no usable track"},
		WithWritableProbe(func(context.Context, string) error { return errors.New("read-only file system") }))

	out, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, RouteSkip, out.Kind)
	assert.Empty(t, h.trans.calls)
}

func TestProcessItem_UnwritableTargetStopsASRBeforeItStarts(t *testing.T) {
	// The ASR leg is paid too: a no_text_source file on a read-only share must
	// be refused before transcription, not after.
	asr := &fakeSpeechTranscriber{available: true}
	h := newItemHarness(t, noTextDecision(),
		WithWritableProbe(func(context.Context, string) error { return errors.New("permission denied") }),
		WithSpeechTranscriber(asr))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})

	require.ErrorIs(t, err, ErrSubtitleTargetNotWritable)
	assert.Empty(t, asr.calls, "speech recognition must not start against an unwritable target")
}

// ─── sub-6-3: extraction queue narration + wait-abort classification ───────

func TestProcessItem_NarratesTheExtractionQueueOnTheProgressStream(t *testing.T) {
	var messages []string
	h := newItemHarness(t, translateDecision("Good morning."),
		WithProgress(func(_ MediaRef, stage PipelineStage, message string) {
			if stage == StageExtracting {
				messages = append(messages, message)
			}
		}))
	h.router.onRoute = func(ctx context.Context) {
		notify := extractWaitNotifierFrom(ctx)
		require.NotNil(t, notify, "ProcessItem must hand the router a ctx carrying the wait notifier")
		notify(3) // queued behind three
		notify(0) // …and let in
	}

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"extracting embedded subtitle track",
		"waiting for the extraction slot (3 ahead)",
		// CR M2: the queue notice is RETRACTED once the slot is taken, or it
		// would sit there for the whole extraction that follows it.
		"extracting embedded subtitle track",
	}, messages)
}

func TestProcessItem_WaitAbortIsRecordedAsCancelledNotAsAFileFailure(t *testing.T) {
	h := newItemHarness(t, RouteDecision{})
	h.router.err = fmt.Errorf("subtitle route: extract x: %w: m.mkv (queued behind 2): %w",
		ErrSubtitleExtractWaitAborted, context.DeadlineExceeded)

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.Error(t, err)

	run := h.runs.updated[len(h.runs.updated)-1]
	assert.Equal(t, models.SubtitleRunFailed, run.Status)
	assert.True(t, strings.HasPrefix(run.ErrorMessage, CancelledRunPrefix),
		"a deadline that fired while queued says nothing about the file — it must not count toward parking: %q", run.ErrorMessage)
	assert.Contains(t, run.ErrorMessage, "SUBTITLE_EXTRACT_WAIT_ABORTED")
}

// ─── sub-6-8a AC #4/#6: the per-run model reaches the row and the cache ────

func TestProcessItem_RecordsTheModelTheCallerChose(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref,
		ProcessItemOptions{ModelID: "claude-haiku-4-5"})
	require.NoError(t, err)

	require.NotEmpty(t, h.runs.created)
	assert.Equal(t, "claude-haiku-4-5", h.runs.created[0].ModelID,
		"the run row must name the model that did the work, not the deployment default")
}

func TestProcessItem_ChosenModelReachesTheProviderThroughTheContext(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	var seen []string
	inner := h.trans.fn
	h.trans.fn = nil
	h.pipeline.translator = &modelSpyTranslator{inner: &fakeTranslator{fn: inner}, seen: &seen}

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref,
		ProcessItemOptions{ModelID: "claude-haiku-4-5"})
	require.NoError(t, err)

	require.NotEmpty(t, seen)
	for _, got := range seen {
		assert.Equal(t, "claude-haiku-4-5", got,
			"the provider holder picks the client off the ctx — nothing in between passes a model parameter")
	}
}

func TestProcessItem_NoChoiceLeavesTheDeploymentDefaultInPlace(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	var seen []string
	inner := h.trans.fn
	h.trans.fn = nil
	h.pipeline.translator = &modelSpyTranslator{inner: &fakeTranslator{fn: inner}, seen: &seen}

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)

	require.NotEmpty(t, seen)
	assert.Empty(t, seen[0], "an unset choice must not pin anything — the holder then applies its own default")
	assert.Equal(t, "claude-haiku-4-5", h.runs.created[0].ModelID,
		"…and the run row records the pipeline's configured model source (WithModelID in this harness)")
}

func TestProcessItem_SwitchingModelDoesNotReuseTheOtherModelsTranslations(t *testing.T) {
	// AC #6: the segment cache key carries the model, so a Haiku translation
	// must never be served to a Sonnet run — the user paid for the better
	// model and must get it.
	source := cues("Good morning.", "See you later.")
	decision := RouteDecision{
		Kind:            RouteTranslate,
		Track:           &ExtractedTrack{StreamIndex: 2, Language: "eng", Blocks: source},
		DetectedVariant: LangUndetermined,
	}
	h := newItemHarness(t, decision)

	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{ModelID: "claude-haiku-4-5"})
	require.NoError(t, err)
	firstCalls := len(h.trans.calls)
	require.NotZero(t, firstCalls)

	// Same item, same cues, different model: every cue must be re-translated.
	h.trans.calls = nil
	_, err = h.pipeline.ProcessItem(context.Background(), h.ref,
		ProcessItemOptions{Force: true, ModelID: "claude-sonnet-5"})
	require.NoError(t, err)

	require.NotEmpty(t, h.trans.calls, "a model switch must not silently reuse the cheaper model's output")
	var sent int
	for _, c := range h.trans.calls {
		sent += len(c.blocks)
	}
	assert.Equal(t, len(source), sent, "every cue is re-sent under the new model")
}

// modelSpyTranslator records the ctx-pinned model each chunk request ran
// under — the value the provider holder would key its client on.
type modelSpyTranslator struct {
	inner *fakeTranslator
	seen  *[]string
}

func (m *modelSpyTranslator) TranslateChunk(ctx context.Context, sys []ai.SystemBlock, contextBlocks, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, map[string]string, ai.CompletionUsage, error) {
	*m.seen = append(*m.seen, ai.ModelIDFromContext(ctx))
	return m.inner.TranslateChunk(ctx, sys, contextBlocks, blocks)
}
