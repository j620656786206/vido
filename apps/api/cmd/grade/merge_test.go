package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/eval"
)

func sampleReport(grade string) eval.Report {
	return eval.Report{
		ModelID: "claude-opus-4-8", SampleVersion: eval.SampleVersion, Grade: grade,
		PromptVersion: "m1-v3+zh-tw-lex-1+standard", LexiconVersion: "zh-tw-lex-1",
		JudgeModel: eval.DefaultJudgeModel, JudgeVersion: eval.JudgeVersion,
		ZeroRate: 0.02, NaturalRate: 0.75, CostUSD: 0.21, Cues: 200,
		GradedAt: time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC), Incomplete: "",
	}
}

func TestMergeFlagsSkipModelValidation(t *testing.T) {
	cfg, err := parseFlags([]string{"--merge", "r.json"})
	if err != nil || cfg.mergePath != "r.json" || cfg.ratingsPath != defaultRatingsPath {
		t.Fatalf("merge mode needs no --model: cfg=%+v err=%v", cfg, err)
	}
}

func TestMergeReport_WritesGoldenRowAndRefusesIncomplete(t *testing.T) {
	dir := t.TempDir()
	ratings := filepath.Join(dir, "model_ratings.json")
	seed, _ := ai.MergeRating(mustEmbedded(t), goldenSeed())
	if err := os.WriteFile(ratings, seed, 0o644); err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(dir, "opus.json")
	writeJSON(t, report, sampleReport("A"))

	row, err := mergeReport(report, ratings)
	if err != nil {
		t.Fatal(err)
	}
	if row.Source != ai.RatingSourceGolden || row.GradedAt != "2026-10-01T03:00:00Z" || row.SampleVersion != eval.SampleVersion {
		t.Fatalf("row: %+v", row)
	}
	out, _ := os.ReadFile(ratings)
	file, err := ai.ParseRatings(out)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, r := range file.Ratings {
		if r.ModelID == "claude-opus-4-8" && r.Source == ai.RatingSourceGolden && r.Grade == "A" {
			found = true
		}
	}
	if !found {
		t.Fatalf("golden row not written: %s", out)
	}

	writeJSON(t, report, sampleReport(eval.GradeIncomplete))
	if _, err := mergeReport(report, ratings); err == nil || !strings.Contains(err.Error(), "not mergeable") {
		t.Fatalf("incomplete must be refused, got %v", err)
	}
	stale := sampleReport("A")
	stale.SampleVersion = "golden-v0"
	writeJSON(t, report, stale)
	if _, err := mergeReport(report, ratings); err == nil || !strings.Contains(err.Error(), "re-run") {
		t.Fatalf("a report from another sample must be refused, got %v", err)
	}
}

func goldenSeed() ai.ModelRating {
	return ai.ModelRating{
		ModelID: "claude-sonnet-4-6", Grade: "B", ZeroRate: 0.03, NaturalRate: 0.65, Cues: 200,
		PromptVersion: "m1-v3+zh-tw-lex-1+standard", JudgeModel: "claude-sonnet-5", JudgeVersion: "judge-v1",
		GradedAt: "2026-10-01T00:00:00Z", Source: ai.RatingSourceGolden, SampleVersion: "golden-v1",
	}
}

func mustEmbedded(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", defaultRatingsPath))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, _ := json.Marshal(v)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// gemini-translation-dispatch: a hosted model graded through its
// OpenAI-compatible endpoint is published under its real id; an Ollama model
// (no catalog provider) is still refused.
func TestMergeReport_LocalPrefixedCatalogModelIsPublishedUnderItsRealID(t *testing.T) {
	rep := sampleReport("A")
	rep.ModelID = "local:gemini-3.5-flash-lite"
	rep.BatchSize = prompts.SubtitleTranslatorBatchSizeFor("gemini-3.5-flash-lite")
	row, err := ratingFromReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if row.ModelID != "gemini-3.5-flash-lite" {
		t.Fatalf("model id: %q", row.ModelID)
	}

	rep.ModelID = "local:hy-mt2-7b"
	if _, err := ratingFromReport(rep); err == nil || !strings.Contains(err.Error(), "local model") {
		t.Fatalf("an uncatalogued local model must stay unpublishable, got %v", err)
	}
	rep.ModelID = "local:claude-sonnet-5"
	if _, err := ratingFromReport(rep); err == nil {
		t.Fatal("a Claude id behind a local endpoint is not the Claude the app dispatches to")
	}
}

func TestMergeReport_RefusesABatchSizeExperiment(t *testing.T) {
	rep := sampleReport("A")
	rep.BatchSize = 50
	if _, err := ratingFromReport(rep); err == nil || !strings.Contains(err.Error(), "experiment") {
		t.Fatalf("a batch-50 report must not be published, got %v", err)
	}
	rep.BatchSize = prompts.SubtitleTranslatorBatchSize
	if _, err := ratingFromReport(rep); err != nil {
		t.Fatalf("a report at the app's batch size stays mergeable: %v", err)
	}
}

// disc-2026-10-translation-batch-size-default: "the app's batch" is per model
// — a Sonnet report must be measured at 50, an old batch-10 one is refused.
func TestMergeReport_AppBatchIsPerModel(t *testing.T) {
	rep := sampleReport("A")
	rep.ModelID = "gemini-3.5-flash-lite"
	rep.BatchSize = 20
	if _, err := ratingFromReport(rep); err != nil {
		t.Fatalf("Flash-Lite at its app batch must merge: %v", err)
	}
	rep.BatchSize = 0 // legacy report, measured at 10
	if _, err := ratingFromReport(rep); err == nil {
		t.Fatal("a Flash-Lite report measured at 10 no longer describes how the app translates")
	}
}
