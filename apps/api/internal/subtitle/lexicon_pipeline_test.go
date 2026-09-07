package subtitle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
)

// sub-7-4 AC #2: the Taiwan lexicon rides the final pass after OpenCC — and
// mainland-produced content is left alone (PRD rule, same as OpenCC).

func TestTranslateTrack_LexiconRewritesMainlandTermsAfterOpenCC(t *testing.T) {
	tr := &fakeTranslator{
		fn: func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			// The model wrote Traditional characters with mainland vocabulary —
			// exactly what OpenCC cannot see.
			return map[int]string{blocks[0].Index: "我在看電視頻道上的視頻，質量很好"}, ai.CompletionUsage{}, nil
		},
	}
	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(cues("I'm watching a video on the TV channel, great quality")), richContext())
	require.NoError(t, err)
	assert.Equal(t, "我在看電視頻道上的影片，品質很好", res.Blocks[0].Text,
		"視頻 → 影片 and 質量 → 品質, but 電視頻道 is left alone")
}

func TestTranslateTrack_LexiconSkipsMainlandProducedContent(t *testing.T) {
	tr := &fakeTranslator{
		fn: func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			return map[int]string{blocks[0].Index: "這個視頻質量很好"}, ai.CompletionUsage{}, nil
		},
	}
	tctx := richContext()
	tctx.Countries = []string{"CN"}
	res, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(cues("Great video quality")), tctx)
	require.NoError(t, err)
	assert.Equal(t, "這個視頻質量很好", res.Blocks[0].Text, "CN content keeps its own vocabulary")
}

// sub-7-4 AC #3: the level reaches the prompt AND the version tuple.

func TestTranslateTrack_LocalizationLevelSelectsTheStyleSection(t *testing.T) {
	tr := &fakeTranslator{
		fn: func(_ int, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error) {
			return map[int]string{blocks[0].Index: "早安"}, ai.CompletionUsage{}, nil
		},
	}
	tctx := richContext()
	tctx.LocalizationLevel = prompts.LocalizationOTT
	_, err := NewPipeline(tr, &recordingConverter{}, nil).
		TranslateTrack(context.Background(), trackOf(cues("Good morning.")), tctx)
	require.NoError(t, err)
	sys := tr.calls[0].sys
	assert.Contains(t, sys[0].Text, "## Localization style (ott)")
	assert.NotContains(t, sys[0].Text, "## Localization style (standard)")
	assert.Contains(t, sys[0].Text, "- Life360 → Life360", "global lexicon terms live in the invariant block")
	assert.NotContains(t, sys[1].Text, "Life360", "…and not in the per-show block")
}

func TestProcessItem_LocalizationLevelSourceFeedsTheVersion(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."),
		WithLocalizationLevelSource(func(context.Context) prompts.LocalizationLevel { return prompts.LocalizationOTT }))
	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)
	assert.Equal(t, "m1-v3+zh-tw-lex-1+ott", h.runs.lastUpdate(t).PromptVersion,
		"the level rides PromptVersion so an ott cache entry is never served to a literal viewer")
}

func TestProcessItem_NoLocalizationSourceIsTheDefaultLevel(t *testing.T) {
	h := newItemHarness(t, translateDecision("Good morning."))
	_, err := h.pipeline.ProcessItem(context.Background(), h.ref, ProcessItemOptions{})
	require.NoError(t, err)
	assert.Equal(t, prompts.PromptVersionFor(prompts.LocalizationStandard), h.runs.lastUpdate(t).PromptVersion)
}

func TestRunVersion_ChangesWithLevelAndLexicon(t *testing.T) {
	p := NewPipeline(&fakeTranslator{}, &recordingConverter{}, nil, WithModelID("m"))
	a := richContext()
	b := richContext()
	b.LocalizationLevel = prompts.LocalizationLiteral
	va, vb := p.runVersion(context.Background(), a), p.runVersion(context.Background(), b)
	assert.NotEqual(t, va.PromptVersion, vb.PromptVersion)
	assert.Equal(t, va.MetadataHash, vb.MetadataHash, "the level is NOT metadata")
	assert.Equal(t, va.GlossaryVersion, vb.GlossaryVersion)
}
