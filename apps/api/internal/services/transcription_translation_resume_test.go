package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/segkey"
)

// ─── disc-2026-09-generation-resume-a AC #3/#5, service seam ──────────────
//
// The run — not the translation service — is what knows the film's metadata,
// its glossary, the prompt style and the model. These tests pin that it hands
// all four down, so a cached cue can only be served back to a run that would
// have asked the model the same question.

func multiCueSRT(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "%d\n00:00:%02d,000 --> 00:00:%02d,000\nLine %d\n\n", i, i, i+1, i)
	}
	return b.String()
}

// fakeTranslationOpenCC converts the two simplified characters the fixtures
// use, and remembers what it was asked to convert.
type fakeTranslationOpenCC struct {
	seen []string
}

func (f *fakeTranslationOpenCC) IsAvailable() bool { return true }
func (f *fakeTranslationOpenCC) ConvertS2TWP(content []byte) ([]byte, error) {
	f.seen = append(f.seen, string(content))
	return []byte(strings.NewReplacer("软件", "軟體", "华", "華").Replace(string(content))), nil
}

func resumeWiredService(t *testing.T, completer ai.TextCompleter, store SegmentStore) (*TranscriptionService, *fakeTranslationOpenCC) {
	t.Helper()
	svc := newWriterWiredService(t, completer, &fakeSubtitleWriter{})
	opencc := &fakeTranslationOpenCC{}
	svc.SetOpenCCConverter(opencc)
	svc.SetSegmentStore(store)
	return svc, opencc
}

// A cue served from the cache still goes through OpenCC and the Taiwan lexicon:
// what is stored is the model's RAW output, and the whole assembled SRT is
// converted afterwards — so a hit is byte-identical to a fresh translation.
func TestTranslateSRT_CachedCuesStillGetConverted(t *testing.T) {
	completer := &countingCompleter{}
	store := newMemoryStore()
	svc, opencc := resumeWiredService(t, completer, store)
	tmp := t.TempDir()

	// Seed the cache the way an interrupted run would have left it: the model's
	// raw output, simplified characters and all.
	ctx := context.Background()
	version := svc.translationRunVersion(ctx,
		svc.mediaMetadataFor(ctx, models.SubtitleRunMediaMovie, uuidB, uuidB), nil, prompts.DefaultLocalizationLevel)
	require.NoError(t, store.Set(context.Background(), segkey.SegmentKey("Line 1", version), "这个软件很好", segkey.TTL))

	zhPath, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB,
		multiCueSRT(3), tmp+"/Movie.en.srt", tmp+"/Movie.mkv", tmp, true)

	require.NoError(t, err)
	require.NotEmpty(t, zhPath)
	require.NotEmpty(t, opencc.seen)
	assert.Contains(t, opencc.seen[0], "这个软件很好", "the cached cue reached the converter…")
	converted, rerr := os.ReadFile(zhPath)
	require.NoError(t, rerr)
	assert.Contains(t, string(converted), "軟體", "…and the conversion applied to it, exactly as to a fresh translation")
	assert.NotContains(t, string(converted), "软件", "the simplified form the cache held never reaches the file")
}

// The version a run computes must name what the run actually feeds the model.
// An empty ModelID is the named trap: default-model runs would key differently
// from the same model picked explicitly and every hit rate would halve.
func TestTranslationRunVersion_IsFullyPopulated(t *testing.T) {
	svc := newWriterWiredService(t, &countingCompleter{}, nil)
	md := prompts.MediaMetadata{Title: "Goblet of Fire", Year: 2005, Genres: []string{"Fantasy"}}
	v := svc.translationRunVersion(context.Background(), md, nil, prompts.DefaultLocalizationLevel)
	assert.Equal(t, segkey.MetadataHash(md), v.MetadataHash,
		"same film, same hash as the extract leg would compute — otherwise the two legs never share a cached cue")

	assert.NotEmpty(t, v.MetadataHash)
	assert.NotEmpty(t, v.GlossaryVersion, "an empty show glossary still hashes the built-in lexicon's version")
	assert.Equal(t, prompts.PromptVersionFor(prompts.DefaultLocalizationLevel), v.PromptVersion)
	assert.Equal(t, ai.DefaultClaudeModel, v.ModelID)

	t.Run("the run's chosen model rides the version", func(t *testing.T) {
		got := svc.translationRunVersion(ai.WithModelID(context.Background(), "claude-haiku-4-5"),
			md, nil, prompts.DefaultLocalizationLevel)
		assert.Equal(t, "claude-haiku-4-5", got.ModelID)
		assert.NotEqual(t, v.ModelID, got.ModelID)
		assert.NotEqual(t, segkey.SegmentKey("Line 1", v), segkey.SegmentKey("Line 1", got),
			"switching model must re-translate, never serve the other model's rendering")
	})
}

// No store wired (the pre-story default) must translate exactly as before.
func TestTranslateSRT_WithoutAStoreIsUnchanged(t *testing.T) {
	completer := &countingCompleter{}
	svc, _ := resumeWiredService(t, completer, nil)
	tmp := t.TempDir()

	_, _, err := svc.translateAndPersist(context.Background(), "job-1", models.SubtitleRunMediaMovie, uuidB,
		multiCueSRT(25), tmp+"/Movie.en.srt", tmp+"/Movie.mkv", tmp, true)

	require.NoError(t, err)
	assert.Equal(t, 3, completer.count(), "25 cues = 3 batches, none of them skipped")
}
