package subtitle

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/services"
)

// ─── Fakes ─────────────────────────────────────────────────────────────────

type translatorCall struct {
	sys           []ai.SystemBlock
	contextBlocks []prompts.SubtitleTranslatorBlock
	blocks        []prompts.SubtitleTranslatorBlock
}

// indexes returns the cue Index values this call asked the LLM to translate.
func (c translatorCall) indexes() []int {
	out := make([]int, 0, len(c.blocks))
	for _, b := range c.blocks {
		out = append(out, b.Index)
	}
	return out
}

// fakeTranslator answers each TranslateChunk from fn, which receives the
// 1-based call number and the requested blocks. order (when set) is the shared
// call-order log the P4 order proof asserts on.
type fakeTranslator struct {
	fn    func(call int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error)
	calls []translatorCall
	order *[]string
}

func (f *fakeTranslator) TranslateChunk(_ context.Context, sys []ai.SystemBlock, contextBlocks, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
	f.calls = append(f.calls, translatorCall{sys: sys, contextBlocks: contextBlocks, blocks: blocks})
	if f.order != nil {
		*f.order = append(*f.order, "translate")
	}
	return f.fn(len(f.calls), blocks)
}

// s2twpFake is a char-level stand-in for OpenCC s2twp: enough of a real
// conversion that "if the converter had run first, the leak would have been
// silently converted away" is demonstrable rather than asserted.
var s2twpFake = strings.NewReplacer("这", "這", "个", "個", "软", "軟", "件", "件")

// recordingConverter records every cue text handed to OpenCC and the order it
// ran in relative to the LLM calls.
type recordingConverter struct {
	inputs []string
	order  *[]string
	err    error
}

func (c *recordingConverter) ConvertS2TWP(content []byte) ([]byte, error) {
	c.inputs = append(c.inputs, string(content))
	if c.order != nil {
		*c.order = append(*c.order, "convert")
	}
	if c.err != nil {
		return content, c.err
	}
	return []byte(s2twpFake.Replace(string(content))), nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────

// cues builds a source track, one cue per text, numbered from 1 with
// second-aligned timestamps.
func cues(texts ...string) []SubtitleBlock {
	out := make([]SubtitleBlock, 0, len(texts))
	for i, text := range texts {
		out = append(out, SubtitleBlock{
			Index: i + 1,
			Start: fmt.Sprintf("00:00:%02d,000", i),
			End:   fmt.Sprintf("00:00:%02d,000", i+1),
			Text:  text,
		})
	}
	return out
}

func trackOf(blocks []SubtitleBlock) *ExtractedTrack {
	return &ExtractedTrack{StreamIndex: 2, Language: "eng", Codec: "subrip", Blocks: blocks}
}

// answer builds a per-index response covering exactly the requested blocks,
// taking each cue's text from texts (keyed by cue Index).
func answer(blocks []prompts.SubtitleTranslatorBlock, texts map[int]string) map[int]string {
	out := make(map[int]string, len(blocks))
	for _, b := range blocks {
		if text, ok := texts[b.Index]; ok {
			out[b.Index] = text
		}
	}
	return out
}

func indexOfFirst(order []string, want string) int {
	for i, v := range order {
		if v == want {
			return i
		}
	}
	return -1
}

func indexOfLast(order []string, want string) int {
	last := -1
	for i, v := range order {
		if v == want {
			last = i
		}
	}
	return last
}

func countOf(order []string, want string) int {
	n := 0
	for _, v := range order {
		if v == want {
			n++
		}
	}
	return n
}

// ─── NAIL 3 (AC #1) ────────────────────────────────────────────────────────

// TestTranslateTrack_SimplifiedLeakTriggersPerCueRetry is the story's first RED
// test: a chunk comes back with one deliberately-Simplified cue among clean
// ones, and ONLY that cue may be resent (FR16).
func TestTranslateTrack_SimplifiedLeakTriggersPerCueRetry(t *testing.T) {
	source := cues("Good morning.", "This software is great.", "See you later.")

	tr := &fakeTranslator{
		fn: func(call int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			switch call {
			case 1:
				// Cue 2 leaks Simplified characters (这/个 are simplified-only).
				return answer(blocks, map[int]string{
					1: "早安",
					2: "这个软件很好用",
					3: "待會見",
				}), ai.CompletionUsage{}, nil
			default:
				return answer(blocks, map[int]string{2: "這個軟體很好用"}), ai.CompletionUsage{}, nil
			}
		},
	}
	conv := &recordingConverter{}

	res, err := NewPipeline(tr, conv, nil).TranslateTrack(context.Background(), trackOf(source), TranslateContext{})
	require.NoError(t, err)

	require.Len(t, tr.calls, 2, "the Simplified leak must trigger exactly one retry call")
	assert.Equal(t, []int{1, 2, 3}, tr.calls[0].indexes())
	assert.Equal(t, []int{2}, tr.calls[1].indexes(), "only the failed cue may be resent (FR16)")

	require.Len(t, res.Blocks, 3)
	assert.Equal(t, "這個軟體很好用", res.Blocks[1].Text)
	assert.Equal(t, 0, res.StubbornCues)
	assert.Equal(t, 0, Detect([]byte(res.Blocks[1].Text)).SimplifiedCount)
}

// TestTranslateTrack_ConverterRunsOnlyAfterGatePasses is the story's second RED
// test — the P4 order proof. If OpenCC ran before the gate, the Simplified leak
// would be converted away, the retry could never fire, and the gate would be
// dead code.
func TestTranslateTrack_ConverterRunsOnlyAfterGatePasses(t *testing.T) {
	var order []string
	source := cues("Good morning.", "This software is great.", "See you later.")

	tr := &fakeTranslator{
		order: &order,
		fn: func(call int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			if call == 1 {
				return answer(blocks, map[int]string{
					1: "早安",
					2: "这个软件很好用",
					3: "待會見",
				}), ai.CompletionUsage{}, nil
			}
			return answer(blocks, map[int]string{2: "這個軟體很好用"}), ai.CompletionUsage{}, nil
		},
	}
	conv := &recordingConverter{order: &order}

	_, err := NewPipeline(tr, conv, nil).TranslateTrack(context.Background(), trackOf(source), TranslateContext{})
	require.NoError(t, err)

	// The gate observed the leak: a second LLM call happened.
	assert.Equal(t, 2, countOf(order, "translate"), "the gate must observe the leak and retry before any conversion")

	// And OpenCC only ran once every gate had passed.
	firstConvert := indexOfFirst(order, "convert")
	require.NotEqual(t, -1, firstConvert, "the converter must run")
	assert.Greater(t, firstConvert, indexOfLast(order, "translate"),
		"OpenCC must run only after the last gate pass (P4)")

	// The proof that the order matters: OpenCC never saw the leaked text — had
	// it run first it would have silently repaired it and the gate would never
	// have fired.
	for _, in := range conv.inputs {
		assert.Equal(t, 0, Detect([]byte(in)).SimplifiedCount,
			"converter received pre-gate text %q — OpenCC would have masked the leak", in)
	}
}

// ─── Chunking + context window (AC #2, #3) ─────────────────────────────────

// TestTranslateTrack_ChunksAtBatchSizeWithTranslatedContextWindow pins the
// transport unit at SubtitleTranslatorBatchSize and proves the read-only
// context window carries the previous TRANSLATED cues, not the English source.
func TestTranslateTrack_ChunksAtBatchSizeWithTranslatedContextWindow(t *testing.T) {
	texts := make([]string, 0, 25)
	for i := 1; i <= 25; i++ {
		texts = append(texts, fmt.Sprintf("Line %d.", i))
	}
	source := cues(texts...)

	tr := &fakeTranslator{
		fn: func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			out := make(map[int]string, len(blocks))
			for _, b := range blocks {
				out[b.Index] = fmt.Sprintf("第%d句", b.Index)
			}
			return out, ai.CompletionUsage{}, nil
		},
	}

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(source), TranslateContext{})
	require.NoError(t, err)
	require.Len(t, res.Blocks, 25)

	require.Len(t, tr.calls, 3, "25 cues at batch size %d", prompts.SubtitleTranslatorBatchSize)
	assert.Len(t, tr.calls[0].blocks, prompts.SubtitleTranslatorBatchSize)
	assert.Len(t, tr.calls[1].blocks, prompts.SubtitleTranslatorBatchSize)
	assert.Len(t, tr.calls[2].blocks, 5)

	assert.Empty(t, tr.calls[0].contextBlocks, "the first chunk has nothing behind it")
	assert.Equal(t, []prompts.SubtitleTranslatorBlock{
		{Index: 6, Text: "第6句"},
		{Index: 7, Text: "第7句"},
		{Index: 8, Text: "第8句"},
		{Index: 9, Text: "第9句"},
		{Index: 10, Text: "第10句"},
	}, tr.calls[1].contextBlocks, "context is the previous %d TRANSLATED cues", prompts.SubtitleTranslatorContextWindow)
}

// ─── Stubborn-cue policy (AC #2) ───────────────────────────────────────────

// alwaysEmptyFor answers every cue except the given index, which always comes
// back empty — a cue no number of retries can rescue.
func alwaysEmptyFor(stubbornIndex int) func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
	return func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
		out := make(map[int]string, len(blocks))
		for _, b := range blocks {
			if b.Index == stubbornIndex {
				out[b.Index] = "   "
				continue
			}
			out[b.Index] = fmt.Sprintf("第%d句", b.Index)
		}
		return out, ai.CompletionUsage{}, nil
	}
}

func TestTranslateTrack_StubbornCueUnderCeilingKeepsEnglish(t *testing.T) {
	texts := make([]string, 0, 20)
	for i := 1; i <= 20; i++ {
		texts = append(texts, fmt.Sprintf("Line %d.", i))
	}
	source := cues(texts...)

	tr := &fakeTranslator{fn: alwaysEmptyFor(3)}

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(source), TranslateContext{})
	require.NoError(t, err, "1 stubborn cue in 20 is exactly the 5%% ceiling, not over it")

	assert.Equal(t, 1, res.StubbornCues)
	assert.Equal(t, "Line 3.", res.Blocks[2].Text, "a stubborn cue keeps its English original (NFR-R1)")
	assert.Equal(t, "第4句", res.Blocks[3].Text, "its neighbours are unaffected")

	// chunk 1: initial + 2 quality retries; chunk 2: one clean call.
	require.Len(t, tr.calls, 4)
	assert.Equal(t, []int{3}, tr.calls[1].indexes())
	assert.Equal(t, []int{3}, tr.calls[2].indexes(), "quality retries are capped at 2 per cue")
	assert.Len(t, tr.calls[3].indexes(), 10, "the second chunk is untouched by the first chunk's retries")
}

func TestTranslateTrack_StubbornCuesOverCeilingFailTheItem(t *testing.T) {
	texts := make([]string, 0, 10)
	for i := 1; i <= 10; i++ {
		texts = append(texts, fmt.Sprintf("Line %d.", i))
	}
	source := cues(texts...)

	tr := &fakeTranslator{fn: alwaysEmptyFor(3)}

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(source), TranslateContext{})

	require.Error(t, err, "1 stubborn cue in 10 is 10%%, over the 5%% ceiling")
	assert.ErrorIs(t, err, ErrSubtitleTranslateFailed)
	assert.Nil(t, res, "a track that would ship part-English ships nothing")
}

// ─── Usage aggregation (AC #2) ─────────────────────────────────────────────

func TestTranslateTrack_UsageAggregatesAcrossChunksAndRetries(t *testing.T) {
	texts := make([]string, 0, 15)
	for i := 1; i <= 15; i++ {
		texts = append(texts, fmt.Sprintf("Line %d.", i))
	}
	source := cues(texts...)

	tr := &fakeTranslator{
		fn: func(call int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			out := make(map[int]string, len(blocks))
			for _, b := range blocks {
				// Cue 2 leaks Simplified on the first call only, forcing one retry.
				if b.Index == 2 && call == 1 {
					out[b.Index] = "这个软件很好用"
					continue
				}
				out[b.Index] = fmt.Sprintf("第%d句", b.Index)
			}
			return out, ai.CompletionUsage{
				InputTokens: 100, OutputTokens: 10,
				CacheCreationInputTokens: 5, CacheReadInputTokens: 1,
			}, nil
		},
	}

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(source), TranslateContext{})
	require.NoError(t, err)

	// 2 chunks + 1 quality retry = 3 billed calls.
	require.Len(t, tr.calls, 3)
	assert.Equal(t, ai.CompletionUsage{
		InputTokens: 300, OutputTokens: 30,
		CacheCreationInputTokens: 15, CacheReadInputTokens: 3,
	}, res.Usage, "retries are billed too and must be counted")
}

// TestTranslateTrack_TranslatorErrorWrapsSentinel — an unrecoverable transport
// error fails the stage with the sub-1-3 sentinel, cause chained.
func TestTranslateTrack_TranslatorErrorWrapsSentinel(t *testing.T) {
	boom := errors.New("provider exploded")
	tr := &fakeTranslator{
		fn: func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			return nil, ai.CompletionUsage{}, boom
		},
	}

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(cues("Good morning.")), TranslateContext{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleTranslateFailed, "unrecoverable LLM errors carry the sub-1-3 sentinel")
	assert.ErrorIs(t, err, boom, "the cause is chained, not flattened (Rule 13)")
	assert.Nil(t, res)
}

// ─── FR17 invariant (AC #3) ────────────────────────────────────────────────

// TestTranslateTrack_MutatedTrackTripsTimestampInvariant drives the invariant
// through the real stage: the fake mutates the routed track's timestamps
// mid-run, which is exactly the class of desync FR17 exists to catch.
func TestTranslateTrack_MutatedTrackTripsTimestampInvariant(t *testing.T) {
	track := trackOf(cues("Good morning.", "See you later."))

	tr := &fakeTranslator{
		fn: func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			track.Blocks[1].Start = "99:59:59,999" // desync injected underneath the stage
			out := make(map[int]string, len(blocks))
			for _, b := range blocks {
				out[b.Index] = fmt.Sprintf("第%d句", b.Index)
			}
			return out, ai.CompletionUsage{}, nil
		},
	}

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), track, TranslateContext{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleTimestampMismatch)
	assert.Nil(t, res)
}

func TestCheckTimestampInvariant(t *testing.T) {
	source := cues("A", "B")

	tests := []struct {
		name       string
		translated []SubtitleBlock
		wantErr    bool
	}{
		{
			name:       "identical identity passes",
			translated: []SubtitleBlock{{Index: 1, Start: source[0].Start, End: source[0].End, Text: "甲"}, {Index: 2, Start: source[1].Start, End: source[1].End, Text: "乙"}},
		},
		{
			name:       "cue dropped",
			translated: []SubtitleBlock{{Index: 1, Start: source[0].Start, End: source[0].End, Text: "甲"}},
			wantErr:    true,
		},
		{
			name:       "index renumbered",
			translated: []SubtitleBlock{{Index: 1, Start: source[0].Start, End: source[0].End, Text: "甲"}, {Index: 3, Start: source[1].Start, End: source[1].End, Text: "乙"}},
			wantErr:    true,
		},
		{
			name:       "start shifted",
			translated: []SubtitleBlock{{Index: 1, Start: "00:00:05,000", End: source[0].End, Text: "甲"}, {Index: 2, Start: source[1].Start, End: source[1].End, Text: "乙"}},
			wantErr:    true,
		},
		{
			name:       "end shifted",
			translated: []SubtitleBlock{{Index: 1, Start: source[0].Start, End: source[0].End, Text: "甲"}, {Index: 2, Start: source[1].Start, End: "00:09:00,000", Text: "乙"}},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkTimestampInvariant(source, tt.translated)
			if !tt.wantErr {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrSubtitleTimestampMismatch)
		})
	}
}

// TestTranslateTrack_PreservesCueIdentity is the happy-path half of FR17.
func TestTranslateTrack_PreservesCueIdentity(t *testing.T) {
	source := []SubtitleBlock{
		{Index: 4, Start: "00:00:01,000", End: "00:00:02,500", Text: "Good morning."},
		{Index: 9, Start: "00:01:11,120", End: "00:01:13,000", Text: "See you later."},
	}

	tr := &fakeTranslator{
		fn: func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			out := make(map[int]string, len(blocks))
			for _, b := range blocks {
				out[b.Index] = fmt.Sprintf("第%d句", b.Index)
			}
			return out, ai.CompletionUsage{}, nil
		},
	}

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(source), TranslateContext{})
	require.NoError(t, err)

	require.Len(t, res.Blocks, 2)
	for i, got := range res.Blocks {
		assert.Equal(t, source[i].Index, got.Index)
		assert.Equal(t, source[i].Start, got.Start)
		assert.Equal(t, source[i].End, got.End)
	}
	assert.Equal(t, []int{4, 9}, tr.calls[0].indexes(), "SDH gaps in the numbering survive into the prompt (P1/P7)")
}

// ─── System blocks (AC #4) ─────────────────────────────────────────────────

func TestTranslateTrack_SystemBlocksAreStableFirstAndUncached(t *testing.T) {
	tctx := TranslateContext{
		Title:     "怪奇物語",
		Year:      2016,
		Genres:    []string{"Drama"},
		Cast:      []string{"Winona Ryder"},
		Countries: []string{"US"},
		Glossary:  []prompts.GlossaryEntry{{Source: "Demogorgon", Target: "魔王獸"}},
	}

	tr := &fakeTranslator{
		fn: func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			return map[int]string{blocks[0].Index: "早安"}, ai.CompletionUsage{}, nil
		},
	}

	_, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(cues("Good morning.")), tctx)
	require.NoError(t, err)

	sys := tr.calls[0].sys
	require.Len(t, sys, 2)
	assert.Equal(t, prompts.SubtitleTranslatorSystemPrompt, sys[0].Text, "the most stable block comes first")
	assert.Contains(t, sys[1].Text, "怪奇物語")
	assert.Contains(t, sys[1].Text, "Demogorgon → 魔王獸")
	assert.Less(t, strings.Index(sys[1].Text, "Media context"), strings.Index(sys[1].Text, "Glossary"))

	for i, b := range sys {
		assert.Equal(t, ai.CacheTTLNone, b.CacheTTL,
			"block %d must not be a cache breakpoint — cache policy is sub-1-5b's", i)
	}
}

func TestTranslateTrack_ZeroContextEmitsOnlyTheStablePrompt(t *testing.T) {
	tr := &fakeTranslator{
		fn: func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			return map[int]string{blocks[0].Index: "早安"}, ai.CompletionUsage{}, nil
		},
	}

	_, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(cues("Good morning.")), TranslateContext{})
	require.NoError(t, err)

	require.Len(t, tr.calls[0].sys, 1, "no metadata and no glossary means no per-show block at all")
	assert.Equal(t, prompts.SubtitleTranslatorSystemPrompt, tr.calls[0].sys[0].Text)
}

// ─── Degradation + guards ──────────────────────────────────────────────────

func TestTranslateTrack_ConverterFailureDeliversGatedText(t *testing.T) {
	tr := &fakeTranslator{
		fn: func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			return map[int]string{blocks[0].Index: "早安"}, ai.CompletionUsage{}, nil
		},
	}
	conv := &recordingConverter{err: errors.New("opencc unavailable")}

	res, err := NewPipeline(tr, conv, nil).
		TranslateTrack(context.Background(), trackOf(cues("Good morning.")), TranslateContext{})

	require.NoError(t, err, "OpenCC is a polish pass, not the correctness barrier — it must not fail the item")
	assert.Equal(t, "早安", res.Blocks[0].Text)
}

// TestNewPipeline_NilPortsPanic — both ports are wiring-time invariants. A nil
// converter in particular would silently void FR15's Traditional-script
// guarantee, so construction must fail loudly, not degrade (sub-1-5a CR M2).
func TestNewPipeline_NilPortsPanic(t *testing.T) {
	assert.Panics(t, func() { NewPipeline(nil, &recordingConverter{}, nil) })
	assert.Panics(t, func() { NewPipeline(&fakeTranslator{}, nil, nil) })
}

func TestTranslateTrack_RejectsEmptyTrack(t *testing.T) {
	p := NewPipeline(&fakeTranslator{}, &recordingConverter{}, nil)

	for _, track := range []*ExtractedTrack{nil, trackOf(nil)} {
		res, err := p.TranslateTrack(context.Background(), track, TranslateContext{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrSubtitleTranslateFailed)
		assert.Nil(t, res)
	}
}

// ─── Port conformance (AC #5) ──────────────────────────────────────────────

// The stage is wired to the real implementations by sub-1-6; these compile-time
// assertions are what keeps the narrow ports honest until then. They live in
// the test file so pipeline.go itself keeps zero coupling to either concrete
// type.
var (
	_ ChunkTranslator  = (*services.TranslationService)(nil)
	_ VariantConverter = (*Converter)(nil)
)

func TestTranslateTrack_HonoursContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tr := &fakeTranslator{
		fn: func(int, []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			t.Fatal("no LLM call may be made after cancellation")
			return nil, ai.CompletionUsage{}, nil
		},
	}

	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(ctx, trackOf(cues("Good morning.")), TranslateContext{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubtitleTranslateFailed, "the run row still classifies it")
	assert.ErrorIs(t, err, context.Canceled, "a shutdown stays distinguishable from a real failure")
	assert.Nil(t, res)
	assert.Empty(t, tr.calls)
}
