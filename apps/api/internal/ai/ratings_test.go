package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── sub-7-8b: model_ratings.json ──────────────────────────────────────────

func goldenRow(id, grade string) ModelRating {
	return ModelRating{
		ModelID: id, Grade: grade, ZeroRate: 0.02, NaturalRate: 0.7, CostUSD: 0.1, Cues: 200,
		PromptVersion: "m1-v3+zh-tw-lex-1+standard", LexiconVersion: "zh-tw-lex-1",
		JudgeModel: "claude-sonnet-5", JudgeVersion: "judge-v1", GradedAt: "2026-10-01T03:00:00Z",
		Source: RatingSourceGolden, SampleVersion: "golden-v1",
	}
}

func TestRatings_EmbeddedFileLoadsAndCarriesEval1(t *testing.T) {
	file, err := ParseRatings(embeddedRatings)
	require.NoError(t, err)
	assert.Equal(t, 1, file.Schema)
	r := Ratings()
	assert.Equal(t, "A", r["claude-sonnet-5"].Grade)
	assert.Equal(t, RatingSourceEval1, r["claude-sonnet-5"].Source)
	assert.Contains(t, RatingNote(r["claude-sonnet-5"]), "eval-1")
}

func TestParseRatings_RejectsWhatMustNotShip(t *testing.T) {
	base := func() RatingsFile {
		return RatingsFile{Schema: 1, SampleVersion: "golden-v1", Ratings: []ModelRating{goldenRow("claude-opus-4-8", "B")}}
	}
	enc := func(f RatingsFile) []byte { b, _ := json.Marshal(f); return b }
	cases := map[string]func(*RatingsFile){
		"incomplete grade":     func(f *RatingsFile) { f.Ratings[0].Grade = "incomplete" },
		"lower-case grade":     func(f *RatingsFile) { f.Ratings[0].Grade = "a" },
		"bad date":             func(f *RatingsFile) { f.Ratings[0].GradedAt = "yesterday" },
		"unknown source":       func(f *RatingsFile) { f.Ratings[0].Source = "vendor" },
		"rate out of range":    func(f *RatingsFile) { f.Ratings[0].ZeroRate = 1.5 },
		"golden without judge": func(f *RatingsFile) { f.Ratings[0].JudgeModel = "" },
		"wrong schema":         func(f *RatingsFile) { f.Schema = 2 },
		"duplicate row": func(f *RatingsFile) {
			f.Ratings = append(f.Ratings, goldenRow("claude-opus-4-8", "A"))
		},
		"eval-1 without note": func(f *RatingsFile) {
			f.Ratings[0].Source = RatingSourceEval1
		},
	}
	for name, mutate := range cases {
		f := base()
		mutate(&f)
		_, err := ParseRatings(enc(f))
		assert.Error(t, err, name)
	}
	_, err := ParseRatings([]byte(`{"schema":1,"ratings":[],"surprise":true}`))
	assert.Error(t, err, "unknown fields are typos in a hand-edited file")
}

func TestIndexRatings_GoldenBeatsEval1(t *testing.T) {
	eval1 := ModelRating{ModelID: "claude-sonnet-5", Grade: "A", Source: RatingSourceEval1, Note: "n", GradedAt: "2026-09-03T00:00:00Z"}
	golden := goldenRow("claude-sonnet-5", "B")
	for _, order := range [][]ModelRating{{eval1, golden}, {golden, eval1}} {
		got := indexRatings(RatingsFile{Schema: 1, Ratings: order})
		assert.Equal(t, RatingSourceGolden, got["claude-sonnet-5"].Source)
		assert.Equal(t, "B", got["claude-sonnet-5"].Grade)
	}
}

func TestRatingNote(t *testing.T) {
	assert.Equal(t, "Vido 實測 2026-10，黃金樣本 v1；裁判 claude-sonnet-5", RatingNote(goldenRow("x", "A")))
	assert.Equal(t, "自訂", RatingNote(ModelRating{Note: "自訂"}))
}

func TestMergeRating_ReplacesSortsAndRefusesIncomplete(t *testing.T) {
	out, err := MergeRating(embeddedRatings, goldenRow("claude-opus-4-8", "A"))
	require.NoError(t, err)
	file, err := ParseRatings(out)
	require.NoError(t, err)
	ids := make([]string, 0, len(file.Ratings))
	for _, r := range file.Ratings {
		ids = append(ids, r.ModelID+"/"+r.Source)
	}
	assert.Equal(t, []string{"claude-haiku-4-5/eval-1", "claude-opus-4-8/golden", "claude-sonnet-5/eval-1"}, ids)
	assert.True(t, strings.HasSuffix(string(out), "}\n"), "pretty JSON with a trailing newline, like a hand-edited file")

	// Second golden run for the same model replaces, never duplicates.
	again, err := MergeRating(out, goldenRow("claude-opus-4-8", "C"))
	require.NoError(t, err)
	file, _ = ParseRatings(again)
	assert.Len(t, file.Ratings, 3)
	for _, r := range file.Ratings {
		if r.ModelID == "claude-opus-4-8" {
			assert.Equal(t, "C", r.Grade)
		}
	}

	// A golden row for an eval-1 model sits beside it and wins at index time.
	both, err := MergeRating(out, goldenRow("claude-sonnet-5", "B"))
	require.NoError(t, err)
	file, _ = ParseRatings(both)
	assert.Len(t, file.Ratings, 4)
	assert.Equal(t, "B", indexRatings(file)["claude-sonnet-5"].Grade)

	bad := goldenRow("claude-opus-4-8", "incomplete")
	_, err = MergeRating(embeddedRatings, bad)
	assert.Error(t, err, "an incomplete run must never be merged")
}
