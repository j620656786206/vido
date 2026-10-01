package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vido/api/internal/ai"
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
